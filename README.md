# Red Discovery

red discovery is one of the derivatives of the RedQueen project, used to implement simple service registration/discovery based on RedQueen

Get red discovery:
```shell
go get github.com/RealFax/red-discovery@latest
```

Note: In go1.21, you need to enable `GOEXPERIMENT=loopvar`, this issue will be solved in the near future

## API

### Register a service
```go
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
```

### Discovery a service
```go
if err := client.Discovery(nil, naming); err != nil {
	// handle discovery error
}
```

### Invoke service
```go
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
```

### Service status listener
```go
listenerID, err := client.UseListener(naming, func(ready bool, conn *discovery.GrpcPoolConn, wg *sync.WaitGroup) {
	// handle listener callback
	// you should defer called wg.Done
})
if err != nil {
	// handle UseListener error
}

// destroy listener by listener id
client.DestroyListener(naming, listenerID)
```

_For more usage, see Example..._
