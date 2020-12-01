# Functional requirements 

Upon receipt of a Request, the connect Server chain element must send, as a client, a corresponding request to 
another Server. For clarity, we will refer to the incoming Request to the server as the 'server Request'.

If the server request.GetConnection.GetId() is not associated to an existing client Connection, a new Connection
must be established by sending a Request to the Server indicated by the clienturl.ClientURL(ctx).

If the server request.GetConnection.GetId() is associated to an existing client Connection, that client Connection needs
to be sent as part of the client request to the server the client connection was received from.

If the server is asked to Close an existing server connection, it should also Close the corresponding client Connection
with the server that client Connection was received from.  Even if the attempt to Close the client connection fails, it should
continue to call down the next.Server(ctx) chain.

If the server is asked to Close a server connection from which it has no corresponding client Connection, it should quietly
return without error.

# Implementation

## connectServer

`connectServer` keeps `connInfos` [connectionInfoMap](https://github.com/networkservicemesh/pkg/networkservice/common/connect/gen.go#25)
mapping incoming server `Connection.ID` to the remote server URL and to the client chain assigned to this URL, and `clients` [clientmap.RefcountMap](https://github.com/networkservicemesh/sdk/blob/master/pkg/tools/clientmap/refcount.go)
mapping remote server URL to a [clienturl.NewClient(...)](https://github.com/networkservicemesh/sdk/blob/master/pkg/networkservice/common/clienturl/client.go)
which handles the instantiation and management of the client connection from the `clienturl.ClientURL(ctx)` of the server
Request. Notably, on every [clienturl.NewClient(...)](https://github.com/networkservicemesh/sdk/blob/master/pkg/networkservice/common/clienturl/client.go)
Close it is deleted from the `clients` map, so eventually its context will be canceled and the corresponding `grpc.ClientConn`
will be closed.

Care is taken to make sure that each client chain results in one increment of the refcount on its creation, and one
decrement when it receives a Close. In this way, we can be sure that `clienturl.NewClient(...)` context is closed after
the last client chain using it has received its Close.

The overall result is that usually, `connectServer` will have no more than one `clienturl.NewClient(...)` per `clientURL`.
It may occasionally have more than one in a transient fashion for the lifetime of one or more Connections.
In all events it will have zero `clienturl.NewClient(...)` for a `clientURL` if it has no server Connections for that
`clientURL`.

## Comments on concurrency characteristics.

Concurrency is primarily managed through type-specific wrappers of [sync.Map](https://golang.org/pkg/sync/#Map):
> The Map type is optimized for two common use cases: (1) when the entry for a given key is only ever written once 
> but read many times, as in caches that only grow, or (2) when multiple goroutines read, write, and overwrite entries 
> for disjoint sets of keys. In these two cases, use of a Map may significantly reduce lock contention compared to a 
> Go map paired with a separate Mutex or RWMutex.

This is precisely our case.

[clientmap.RefcountMap](https://github.com/networkservicemesh/sdk/blob/master/pkg/tools/clientmap/refcount.go#L43) utilizes
a simple atomic refcount, which is also highly performant.
