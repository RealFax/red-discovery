package balancer_test

import (
	"github.com/oops-dev/merchant/pkg/balancer"
	"strconv"
	"testing"
)

var (
	roundRobin = balancer.NewRoundRobin[string, *node]()
)

func init() {
	roundRobin.Append(
		&node{"node-1", "addr1"},
		&node{"node-2", "addr2"},
		&node{"node-3", "addr3"},
	)

	for i := 0; i < 10000; i++ {
		roundRobin.Append(&node{
			name: "node-" + strconv.Itoa(i),
			addr: "address" + strconv.Itoa(i),
		})
	}
}

func TestRoundRobinBalance_Next(t *testing.T) {
	for i := 0; i < 10; i++ {
		n, _ := roundRobin.Next()
		t.Logf("name: %s, addr: %s", n.name, n.addr)
	}
}

func BenchmarkRoundRobinBalance_Next(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = roundRobin.Next()
	}
}

func BenchmarkRoundRobinBalance_Next_Parallel(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = roundRobin.Next()
		}
	})
}
