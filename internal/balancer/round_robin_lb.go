package balancer

import (
	"github.com/pkg/errors"
	"sync/atomic"
)

type roundRobinBalance[K comparable, V any] struct {
	current atomic.Int32
	*loadBalanceStore[K, V]
}

func (b *roundRobinBalance[K, V]) Next() (V, error) {
	size := b.size.Load()
	if size == 0 {
		var empty V
		return empty, errors.New("empty load balance list")
	}
	next := b.current.Add(1) % size
	b.current.CompareAndSwap(b.current.Load(), next)
	return b.nodes[next].Value(), nil
}

func (b *roundRobinBalance[K, V]) Remove(key K) bool {
	if !b.loadBalanceStore.Remove(key) {
		return false
	}
	b.current.Add(-1)
	return true
}

func NewRoundRobin[K comparable, V any]() LoadBalance[K, V] {
	return &roundRobinBalance[K, V]{loadBalanceStore: newLoadBalanceStore[K, V]()}
}
