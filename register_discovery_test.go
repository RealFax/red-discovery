package discovery_test

import (
	discovery "github.com/RealFax/red-discovery"
	"google.golang.org/grpc"
	"sync"
)

func ExampleDiscoveryAndRegister_Discovery() {
	if err := client.Discovery(naming); err != nil {
		// handle discovery error
	}
}

func ExampleDiscoveryAndRegister_Register() {
	endpoint := &discovery.Endpoint{
		ID:          "node-1",
		PeerAddress: "localhost:8080",
		Metadata:    nil,
	}

	// set endpoint ttl
	endpoint.SetTTL(1200)

	// put endpoint metadata
	_ = endpoint.PutMetadata(discovery.NewKVMetadataFromMap(map[string]string{
		"name": "super-node-1",
	}))

	if err := client.Register(naming, endpoint); err != nil {
		// handler register error
	}
}

func ExampleDiscoveryAndRegister_UseListener() {
	listenerID, err := client.UseListener(naming, func(ready bool, conn *grpc.ClientConn, wg *sync.WaitGroup) {
		// handle listener callback
		// you should defer called wg.Done
	})
	if err != nil {
		// handle UseListener error
	}

	// destroy listener by listener id
	client.DestroyListener(naming, listenerID)
}
