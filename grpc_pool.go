package discovery

import (
	"context"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"sync/atomic"
)

var (
	poolSize uint32 = 32
)

func ConnPoolSize() uint32 {
	return atomic.LoadUint32(&poolSize)
}

func SetConnPoolSize(size uint32) {
	atomic.StoreUint32(&poolSize, size)
}

const (
	ConnStateWait int32 = iota
	ConnStateAlloc
	ConnStateClose
)

type GrpcPoolConn struct {
	readonly atomic.Bool
	state    atomic.Int32
	release  func(c *GrpcPoolConn)
	ID       string
	*grpc.ClientConn
}

func (c *GrpcPoolConn) Close() error {
	if c.state.Load() == ConnStateClose {
		return nil
	}

	if c.readonly.Load() {
		return ErrCloseReadonlyConn
	}

	c.state.Store(ConnStateClose)
	return c.ClientConn.Close()
}

func (c *GrpcPoolConn) Release() {
	if c.state.Load() != ConnStateAlloc {
		return
	}
	c.state.Store(ConnStateWait)
	c.release(c)
}

// GrpcPool implements a high-performance GRPC connection pool
type GrpcPool struct {
	state    atomic.Int32
	ctx      context.Context
	dialOpts []grpc.DialOption
	endpoint *Endpoint

	rings chan *GrpcPoolConn
}

func (p *GrpcPool) factory() (*GrpcPoolConn, error) {
	conn, err := grpc.DialContext(
		p.ctx,
		p.endpoint.PeerAddress,
		p.dialOpts...,
	)
	if err != nil {
		return nil, err
	}
	return &GrpcPoolConn{
		state:      atomic.Int32{},
		release:    p.Free,
		ID:         uuid.New().String(),
		ClientConn: conn,
	}, nil
}

func (p *GrpcPool) Alloc(readonly bool) (*GrpcPoolConn, error) {
	if p.state.Load() == ConnStateClose {
		return nil, ErrConnPoolClosed
	}

alloc:
	select {
	case c := <-p.rings:
		if c == nil {
			return nil, ErrConnPoolClosed
		}

		if state := c.state.Load(); state != ConnStateWait {
			goto alloc
		}

		// set conn state
		c.state.Store(ConnStateAlloc)
		c.readonly.Store(readonly)
		return c, nil
	default:
		return p.factory()
	}
}

func (p *GrpcPool) Free(c *GrpcPoolConn) {
	if c.state.Load() != ConnStateAlloc {
		return
	}

	// reset conn state
	c.state.Store(ConnStateWait)
	c.readonly.Store(false)

	select {
	case p.rings <- c: // try pushback conn to rings
		return
	default:
		_ = c.Close()
	}
}

func (p *GrpcPool) Close() error {
	if p.state.Load() == ConnStateClose {
		return ErrConnPoolClosed
	}
	p.state.Store(ConnStateClose)

	for {
		select {
		case c := <-p.rings:
			_ = c.Close()
		default:
			return nil
		}
	}
}

func NewGrpcPool(ctx context.Context, endpoint *Endpoint, dialOpts ...grpc.DialOption) (*GrpcPool, error) {
	var (
		// copy conn pool size
		size = ConnPoolSize()
		pool = &GrpcPool{
			ctx:      ctx,
			dialOpts: dialOpts,
			endpoint: endpoint,
			rings:    make(chan *GrpcPoolConn, size),
		}
	)

	for i := 0; i < int(size); i++ {
		factory, err := pool.factory()
		if err != nil {
			return nil, err
		}
		pool.rings <- factory
	}
	return pool, nil
}
