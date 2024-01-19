package discovery

import (
	"context"
	"github.com/RealFax/RedQueen/client"
	"github.com/RealFax/red-discovery/internal/hack"
	"github.com/RealFax/red-discovery/internal/maputil"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"sync"
	"time"
)

type ListenCallbackFunc func(ready bool, conn *grpc.ClientConn, wg *sync.WaitGroup)

type DiscoveryAndRegister interface {
	// ReleaseDiscovery cancel the discovery of a Naming.
	ReleaseDiscovery(naming string)

	// Discovery a Naming, will open a goroutine to achieve continuous discovery of Naming.
	Discovery(namespace *string, naming string) error

	// Unregister one or more services using an Endpoint ID.
	Unregister(namespace *string, naming string, ids ...string) error

	// Register one or more services with Naming.
	Register(namespace *string, naming string, endpoints ...*Endpoint) error

	// UseListener
	//
	// Monitors whether a Naming is available.
	// When ready is true, it means that there are available services for this Naming.
	// When it is false, it means that there are no services available under the Naming.
	UseListener(naming string, callback ListenCallbackFunc) (string, error)

	// DestroyListener cancel listening to Naming based on the ListenerID returned by UseListener.
	DestroyListener(naming, listenerID string)
}

type discoveryAndRegister struct {
	ctx       context.Context
	dialOpts  []grpc.DialOption
	kvClient  client.KvClient
	services  *maputil.Map[string, Service]                                  // map<naming, Service>
	discovery *maputil.Map[string, context.CancelFunc]                       // map<naming, discoverySignal>
	listener  *maputil.Map[string, *maputil.Map[string, ListenCallbackFunc]] // map<string, map<string, ListenCallbackFunc>>
}

func (r *discoveryAndRegister) notifyStateChange(srv Service) {
	childListener, found := r.listener.Load(srv.Naming())
	if !found {
		return
	}

	var (
		wg      = &sync.WaitGroup{}
		state   = srv.Alive()
		conn, _ = srv.NextAliveConn()
	)

	childListener.Range(func(_ string, fc ListenCallbackFunc) bool {
		wg.Add(1)
		fc(state, conn, wg)
		return true
	})

	go func() {
		wg.Wait()
	}()
}

func (r *discoveryAndRegister) discoveryDaemon(namespace *string, naming string) {
	// double-check-area naming discovery status
	ctx, cancel := context.WithCancel(r.ctx)

	// release naming discovery lock
	// defer cancel()

	// discovery has existed
	if _, ok := r.discovery.LoadOrStore(naming, cancel); ok {
		return
	}

	watcher := client.NewWatcher(
		hack.String2Bytes(naming),
		client.WatchWithPrefix(),
		client.WatchWithNamespace(namespace),
	)
	defer watcher.Close()

	go func() {
		defer func() {
			// prevent variable race
			_cancel, ok := r.discovery.LoadAndDelete(naming)
			if !ok {
				return
			}
			_cancel()
		}()
		// async watch
		if err := r.kvClient.WatchPrefix(ctx, watcher); err != nil {
			return
		}
	}()

	// get watcher notify
	var (
		ok                         bool
		err                        error
		endpointNaming, endpointID string
		value                      *client.WatchValue
		endpoint                   *Endpoint
		srv                        Service
		notify, _                  = watcher.Notify()
	)
	for {
		select {
		case <-ctx.Done():
			return
		case value = <-notify:
			if value == nil {
				continue
			}

			if endpointNaming, endpointID, err = ParseEndpointPath(hack.Bytes2String(value.Key)); err != nil {
				continue
			}

			// trying load exist service, if not found and value not nil then init service
			if srv, ok = r.services.Load(endpointNaming); !ok {
				srv, ok = r.services.LoadOrStore(endpointNaming, NewService(r.ctx, endpointNaming, r.dialOpts...))
				if !ok {
					srv, _ = r.services.Load(endpointNaming)
				}
			}

			// deleted
			if value.Value == nil {
				srv.DelEndpoints(endpointID)

				// notify naming listeners
				r.notifyStateChange(srv)
				continue
			}

			if endpoint, err = ParseEndpoint(value.Value); err != nil {
				continue
			}

			endpoint.lastUpdated = value.Timestamp
			endpoint.SetTTL(value.TTL)

			// update endpoint
			srv.AddEndpoints(endpoint)

			// notify naming listeners
			r.notifyStateChange(srv)
		}
	}
}

func (r *discoveryAndRegister) ReleaseDiscovery(naming string) {
	cancel, ok := r.discovery.LoadAndDelete(naming)
	if !ok {
		return
	}
	cancel()
}

func (r *discoveryAndRegister) Discovery(namespace *string, naming string) error {
	// check discovery status
	if exist := r.discovery.Exist(naming); exist {
		if !r.services.Exist(naming) {
			r.services.LoadOrStore(naming, NewService(r.ctx, naming, r.dialOpts...))
		}
		return ErrDiscoveryHasExist
	}

	go r.discoveryDaemon(namespace, naming)

	values, err := r.kvClient.PrefixScan(
		r.ctx,
		hack.String2Bytes(naming),
		0,
		MaxEndpointSize,
		nil,
		namespace,
	)
	if err != nil {
		return nil
	}

	srv, ok := r.services.Load(naming)
	if !ok {
		srv = NewService(r.ctx, naming, r.dialOpts...)
		r.services.Store(naming, srv)
	}

	var endpoint *Endpoint
	for _, value := range values {
		if endpoint, err = ParseEndpoint(value.Data); err != nil {
			return err
		}
		endpoint.SetTTL(value.TTL)
		endpoint.lastUpdated = time.Now().UnixMilli()
		srv.AddEndpoints(endpoint)
	}

	return nil
}

func (r *discoveryAndRegister) Unregister(namespace *string, naming string, ids ...string) (err error) {
	srv, ok := r.services.Load(naming)
	if !ok {
		return ErrServiceNotExist
	}

	for _, id := range ids {
		if err = r.kvClient.Delete(
			r.ctx,
			hack.String2Bytes(srv.WithEndpointNaming(id)),
			namespace,
		); err != nil {
			return
		}

		// del endpoint from service
		srv.DelEndpoints(id)
	}

	return
}

func (r *discoveryAndRegister) Register(namespace *string, naming string, endpoints ...*Endpoint) (err error) {
	srv, ok := r.services.Load(naming)
	if !ok {
		srv = NewService(r.ctx, naming, r.dialOpts...)
		r.services.Store(naming, srv)
	}

	var endpointOut []byte
	// registered endpoints
	for _, endpoint := range endpoints {
		endpointOut, _ = endpoint.Marshal()
		if err = r.kvClient.Set(
			r.ctx,
			hack.String2Bytes(endpoint.WithNaming(naming)),
			endpointOut,
			endpoint.TTL(),
			namespace,
		); err != nil {
			continue
		}

		// add endpoint to service
		srv.AddEndpoints(endpoint)
	}
	return
}

func (r *discoveryAndRegister) UseListener(naming string, callback ListenCallbackFunc) (string, error) {
	if exist := r.discovery.Exist(naming); !exist {
		return "", ErrShouldDiscoveryFirst
	}

	var (
		found         bool
		childListener *maputil.Map[string, ListenCallbackFunc]
	)
	if childListener, found = r.listener.Load(naming); !found {
		childListener, found = r.listener.LoadOrStore(naming, maputil.New[string, ListenCallbackFunc]())
		if !found {
			childListener, _ = r.listener.Load(naming)
		}
	}

	listenerID := uuid.NewString()
	childListener.Store(listenerID, callback)
	return listenerID, nil
}

func (r *discoveryAndRegister) DestroyListener(naming, listenerID string) {
	if exist := r.discovery.Exist(naming); !exist {
		return
	}

	childListener, found := r.listener.Load(naming)
	if !found {
		return
	}

	childListener.Delete(listenerID)
}

func NewDiscoveryAndRegister(
	ctx context.Context,
	services *maputil.Map[string, Service],
	c *client.Client,
	dialOpts ...grpc.DialOption,
) DiscoveryAndRegister {
	return &discoveryAndRegister{
		ctx:       ctx,
		dialOpts:  dialOpts,
		kvClient:  c,
		services:  services,
		discovery: maputil.New[string, context.CancelFunc](),
		listener:  maputil.New[string, *maputil.Map[string, ListenCallbackFunc]](),
	}
}
