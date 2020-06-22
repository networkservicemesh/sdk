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
target := "callback:{client-authority}"
// If target is not in a list of callback server targets, 
// it will perform grpc.DialContext to connect to passed target
nsmClientGRPC, err := grpc.DialContext(context.Background(), target, callbackServer.WithCallbackDialer(), grpc.WithInsecure())
nsmClient := networkservice.NewNetworkServiceClient(nsmClientGRPC)

resp, _ := nsmClient.Request(context.Background(), &networkservice.NetworkServiceRequest{
    Connection: &networkservice.Connection{
        Id: "qwe",
    },
})
```

### Client identification.

Server could identify clients from passed metadata values, clients could define how to extract
identification from context. Servers adds "callback:" to identity, so then dial with callback 
it is required to add "callback:{client-id}".

```go
// IdentityByAuthority - return identity by :authority
func IdentityByPeerCertificate(ctx context.Context) (string, error) {
	p, ok := peer.FromContext(ctx)
	if !ok {
		err := errors.New("No peer is provided")
		logrus.Error(err)
		return "", err
	}
	tlsInfo, tlsOk := p.AuthInfo.(credentials.TLSInfo)
	if !tlsOk {
		err := errors.New("No TLS info is provided")
		logrus.Error(err)
		return "", err
	}
	commonName := tlsInfo.State.PeerCertificates[0].Subject.CommonName
	return commonName, nil
}
```
