package balancer

import (
	"github.com/RealFax/red-discovery/internal/maputil"
	"slices"
	"sync/atomic"
)

type Node[K comparable, V any] interface {
	Key() K
	Value() V
}

type LoadBalance[K comparable, V any] interface {
	Append(node ...Node[K, V])
	Remove(key K) bool
	Next() (V, error)
}

type loadBalanceStore[K comparable, V any] struct {
	size   atomic.Int32
	filter *maputil.Map[K, struct{}]
	nodes  []Node[K, V]
}

func (s *loadBalanceStore[K, V]) Append(nodes ...Node[K, V]) {
	for i, node := range nodes {
		if _, ok := s.filter.Load(node.Key()); ok {
			nodes = append(nodes[:i], nodes[i+1:]...)
			continue
		}
		s.filter.Store(node.Key(), struct{}{})
	}

	s.nodes = append(s.nodes, nodes...)
	s.size.Add(int32(len(nodes)))
}

func (s *loadBalanceStore[K, V]) Remove(key K) bool {
	if _, ok := s.filter.LoadAndDelete(key); !ok {
		return false
	}

	s.nodes = slices.DeleteFunc(s.nodes, func(n Node[K, V]) bool {
		return n.Key() == key
	})

	s.size.Add(-1)
	return true
}

func newLoadBalanceStore[K comparable, V any]() *loadBalanceStore[K, V] {
	return &loadBalanceStore[K, V]{
		filter: maputil.New[K, struct{}](),
		nodes:  make([]Node[K, V], 0),
	}
}
