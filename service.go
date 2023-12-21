package discovery

import (
	"context"
	"github.com/RealFax/red-discovery/internal/balancer"
	"github.com/RealFax/red-discovery/internal/maputil"
	"google.golang.org/grpc"
	"sync"
	"sync/atomic"
)

type Service interface {
	// Naming returns service naming.
	Naming() string

	// SetNaming set service naming.
	SetNaming(naming string)

	WithEndpointNaming(endpointID string) string

	// Alive returns whether the current service is available.
	Alive() bool

	// AddEndpoints by endpoints slice.
	AddEndpoints(endpoints ...*Endpoint)

	// DelEndpoints by id slice.
	DelEndpoints(ids ...string)

	RangeEndpoints(f func(endpoint *Endpoint) bool)

	// AliveConn returns the grpc connection of all service endpoints on internal.
	//
	// DON'T CLOSE GRPC CONN
	AliveConn() map[string]*GrpcPoolConn

	// NextAliveConn returns the internal grpc connection through the load balancing algorithm.
	//
	// DON"T CLOSE GRPC CONN
	NextAliveConn() (*GrpcPoolConn, error)

	// CloseAliveConn Close internal all grpc conn.
	CloseAliveConn()

	// DialContext realizes the grpc connect function of internal load balancing through round-robin algorithm.
	DialContext(ctx context.Context, opts ...grpc.DialOption) (*grpc.ClientConn, error)

	// DialAll connect to all endpoints.
	DialAll(ctx context.Context, opts ...grpc.DialOption) (map[string]*grpc.ClientConn, error)
}

// simple Service implement
type service struct {
	ctx            context.Context
	dialOpts       []grpc.DialOption
	aliveConnCount atomic.Int64
	naming         *atomic.Pointer[string]
	endpoints      *maputil.Map[string, *Endpoint] // map<id, *Endpoint>
	aliveConn      *maputil.Map[string, *GrpcPool /**grpc.ClientConn*/]
	loadBalance    balancer.LoadBalance[string, *Endpoint]
}

func (s *service) dialEndpoints(endpoints []*Endpoint) {
	wg := sync.WaitGroup{}
	for _, endpoint := range endpoints {
		wg.Add(1)
		go func() {
			defer wg.Done()
			pool, err := NewGrpcPool(s.ctx, endpoint, s.dialOpts...)
			if err != nil {
				s.endpoints.Delete(endpoint.ID)
				return
			}
			s.aliveConn.Store(endpoint.ID, pool)
			s.aliveConnCount.Add(1)
		}()

	}
	wg.Wait()
}

func (s *service) Naming() string {
	return *s.naming.Load()
}

func (s *service) SetNaming(naming string) {
	s.naming.Store(&naming)
}

func (s *service) WithEndpointNaming(endpointID string) string {
	endpoint, ok := s.endpoints.Load(endpointID)
	if !ok {
		return s.Naming()
	}
	return endpoint.WithNaming(s.Naming())
}

func (s *service) Alive() bool {
	count := 0
	s.aliveConn.Range(func(_ string, _ *GrpcPool) bool {
		count++
		return true
	})

	return s.aliveConnCount.Load() != 0
}

func (s *service) AddEndpoints(endpoints ...*Endpoint) {
	var waitDialEndpoints []*Endpoint

	for _, endpoint := range endpoints {
		if e, ok := s.endpoints.Load(endpoint.ID); ok {
			e.lastUpdated = endpoint.lastUpdated
			e.SetTTL(endpoint.TTL())
			continue
		}
		s.endpoints.Store(endpoint.ID, endpoint)
		s.loadBalance.Append(endpoint)
		waitDialEndpoints = append(waitDialEndpoints, endpoint)
	}

	s.dialEndpoints(waitDialEndpoints)
}

func (s *service) DelEndpoints(ids ...string) {
	for _, id := range ids {
		s.endpoints.Delete(id)
		s.loadBalance.Remove(id)
		pool, ok := s.aliveConn.Load(id)
		if !ok {
			continue
		}
		s.aliveConnCount.Add(-1)
		_ = pool.Close()
	}
}

func (s *service) RangeEndpoints(f func(endpoint *Endpoint) bool) {
	s.endpoints.Range(func(_ string, endpoint *Endpoint) bool {
		if endpoint.Expired() {
			return true
		}
		return f(endpoint)
	})
}

func (s *service) AliveConn() map[string]*GrpcPoolConn {
	var (
		err error
		m   = make(map[string]*GrpcPoolConn)
	)
	s.aliveConn.Range(func(key string, value *GrpcPool) bool {
		if m[key], err = value.Alloc(true); err != nil {
			return false
		}
		return true
	})
	return m
}

func (s *service) NextAliveConn() (*GrpcPoolConn, error) {
	endpoint, err := s.loadBalance.Next()
	if err != nil {
		return nil, err
	}
	pool, ok := s.aliveConn.Load(endpoint.ID)
	if !ok {
		return nil, ErrServiceUnreachable
	}
	return pool.Alloc(true)
}

func (s *service) CloseAliveConn() {
	s.aliveConn.Range(func(key string, value *GrpcPool) bool {
		s.aliveConn.Delete(key)
		s.aliveConnCount.Add(-1)
		_ = value.Close()
		return true
	})
}

func (s *service) DialContext(ctx context.Context, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	var (
		err      error
		count    int32
		endpoint *Endpoint
	)

	for {
		endpoint, err = s.loadBalance.Next()
		switch {
		case err != nil:
			return nil, err
		case count >= 3:
			return nil, ErrServiceUnreachable
		case endpoint.Expired():
			count++
			continue
		}
		break
	}

	return grpc.DialContext(ctx, endpoint.PeerAddress, opts...)
}

func (s *service) DialAll(ctx context.Context, opts ...grpc.DialOption) (map[string]*grpc.ClientConn, error) {
	var (
		err   error
		conns = make(map[string]*grpc.ClientConn)
	)

	wg := sync.WaitGroup{}
	s.RangeEndpoints(func(endpoint *Endpoint) bool {
		wg.Add(1)
		go func() {
			defer wg.Done()
			conns[endpoint.ID], err = grpc.DialContext(ctx, endpoint.PeerAddress, opts...)
		}()
		return true
	})
	wg.Wait()

	// error, Close dialed conn
	if err != nil {
		for _, conn := range conns {
			_ = conn.Close()
		}
		return nil, err
	}

	return conns, nil
}

func NewService(ctx context.Context, naming string, dialOpts ...grpc.DialOption) Service {
	// to atomic.Pointer
	_naming := atomic.Pointer[string]{}
	_naming.Store(&naming)

	return &service{
		ctx:         ctx,
		dialOpts:    dialOpts,
		naming:      &_naming,
		endpoints:   maputil.New[string, *Endpoint](),
		aliveConn:   maputil.New[string, *GrpcPool](),
		loadBalance: balancer.NewRoundRobin[string, *Endpoint](),
	}
}
