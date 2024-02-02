package discovery

import (
	"context"
	"github.com/RealFax/RedQueen/client"
	"github.com/RealFax/red-discovery/internal/maputil"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"sync/atomic"
)

var (
	DefaultDialOpts = []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
	}

	namespace atomic.Pointer[string]
)

func Namespace() string {
	if ns := namespace.Load(); ns != nil {
		return *ns
	}
	return ""
}

func SetNamespace(ns string) {
	if ns != "" {
		namespace.Store(&ns)
	}
}

const (
	MaxEndpointSize uint64 = 8192
)

type Client struct {
	ctx      context.Context
	dialOpts []grpc.DialOption
	client   *client.Client
	services *maputil.Map[string, Service]
	DiscoveryAndRegister
}

func (c *Client) Service(naming string) (Service, bool) {
	return c.services.Load(naming)
}

func (c *Client) Close() error {
	c.services.Range(func(key string, value Service) bool {
		c.services.Delete(key)
		value.CloseAliveConn()
		return true
	})
	return c.client.Close()
}

func NewWithClient(ctx context.Context, c *client.Client, dialOpts ...grpc.DialOption) *Client {
	if dialOpts == nil {
		dialOpts = DefaultDialOpts
	}

	services := maputil.New[string, Service]()
	return &Client{
		ctx:                  ctx,
		dialOpts:             dialOpts,
		client:               c,
		services:             services,
		DiscoveryAndRegister: NewDiscoveryAndRegister(ctx, services, c, dialOpts...),
	}
}

func New(ctx context.Context, endpoints []string, dialOpts ...grpc.DialOption) (*Client, error) {
	c, err := client.New(ctx, endpoints, dialOpts...)
	if err != nil {
		return nil, errors.Wrap(err, "dial endpoints")
	}
	return NewWithClient(ctx, c, dialOpts...), nil
}
