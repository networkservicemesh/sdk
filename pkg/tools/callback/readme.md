## Why?

NSM need a server to call client via GRPC, for this purpuse in native way it required to 
have client serve a GRPC server on unix socket or TCP. 
But to simplify communication between client and server a better way is to have 
only one connection between client and server.

And this grpc.callback library is intended to perform required functionality using 
grpc bi-directional streams.

## How?

A Server declared as usual except:

```go
server := grpc.NewServer()
...
// register normal APIs on server
...
// Register callback GRPC endpoint and callback.Server will hold callback clients to be callable.
callbackServer := callback.NewServer(callback.IdentityByAuthority)
//
callback.RegisterCallbackServiceServer(server, callbackServer)
...
_ = server.Serve(listener)
```

And then require to call client.
```go
nsmClientGRPC, _ := callbackServer.NewClient(context.Background(), clientID)
nsmClient := networkservice.NewNetworkServiceClient(nsmClientGRPC)

resp, _ := nsmClient.Request(context.Background(), &networkservice.NetworkServiceRequest{
    Connection: &networkservice.Connection{
        Id: "qwe",
    },
})
```

### Client identification.

Server could identify clients from passed metadata values, default identity is 
using ":authority" passed from client to server.

```go
md, ok := metadata.FromIncomingContext(serverClient.Context())
if !ok {
    logrus.Errorf("Not metadata provided")
}
key := s.identityProvider(md)
```