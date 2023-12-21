package discovery

import (
	"context"
	"github.com/RealFax/RedQueen/client"
	"github.com/RealFax/red-discovery/internal/maputil"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	DefaultDialOpts = []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
	}
)

const (
	MaxEndpointSize uint64 = 8192
)

type Client struct {
	ctx       context.Context
	dialOpts  []grpc.DialOption
	client    *client.Client
	services  *maputil.Map[string, Service]
	Namespace *string

	DiscoveryAndRegister
}

func (c *Client) SetNamespace(namespace string) {
	if namespace == "" {
		c.Namespace = nil
		return
	}
	c.Namespace = &namespace
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
