package discovery_test

import (
	rClient "github.com/RealFax/RedQueen/client"
	discovery "github.com/RealFax/red-discovery"
	"sync"
)

func ExampleDiscoveryAndRegister_Discovery() {
	if err := client.Discovery(nil, naming); err != nil {
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

	if err := client.Register(nil, naming, endpoint); err != nil {
		// handler register error
	}
}

func ExampleDiscoveryAndRegister_UseListener() {
	listenerID, err := client.UseListener(naming, func(ready bool, conn *rClient.BalancerConn, wg *sync.WaitGroup) {
		// handle listener callback
		// you should defer called wg.Done
	})
	if err != nil {
		// handle UseListener error
	}

	// destroy listener by listener id
	client.DestroyListener(naming, listenerID)
}
