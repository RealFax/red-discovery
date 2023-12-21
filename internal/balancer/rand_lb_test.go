package balancer_test

import (
	"github.com/oops-dev/merchant/pkg/balancer"
	"strconv"
	"testing"
)

type node struct {
	name string
	addr string
}

func (n *node) Key() string  { return n.name }
func (n *node) Value() *node { return n }

var (
	random = balancer.NewRandom[string, *node]()
)

func init() {
	random.Append(
		&node{"node-1", "addr1"},
		&node{"node-2", "addr2"},
		&node{"node-3", "addr3"},
	)

	for i := 0; i < 10000; i++ {
		random.Append(&node{
			name: "node-" + strconv.Itoa(i),
			addr: "address" + strconv.Itoa(i),
		})
	}
}

func TestRandomBalance_Next(t *testing.T) {
	for i := 0; i < 10; i++ {
		n, _ := random.Next()
		t.Logf("name: %s, addr: %s", n.name, n.addr)
	}
}

func BenchmarkRandomBalance_Next(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = random.Next()
	}
}
