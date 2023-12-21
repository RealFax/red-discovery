package discovery_test

import (
	"context"
	discovery "github.com/RealFax/red-discovery"
)

const (
	naming = "pkg.discovery.test"
)

var (
	client *discovery.Client
)

func init() {
	var err error
	if client, err = discovery.New(context.Background(), []string{
		"127.0.0.1:5230",
		"127.0.0.1:4230",
		"127.0.0.1:3230",
	}); err != nil {
		panic("init sdr client error, cause: " + err.Error())
	}
}

func ExampleClient_Service() {
	srv, found := client.Service(naming)
	if !found {
		// handle service not found error (discovery not called first)
	}

	conn, err := srv.NextAliveConn()
	if err != nil {
		// handle error
	}

	// invoke grpc api
	// after the call is completed, conn.Release() should be called to release the connection
	conn.Target()
}
