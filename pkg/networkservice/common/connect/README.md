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

# Implementation

## connectServer

`connectServer` keeps `connInfos` [connectionInfoMap](./gen.go) mapping incoming server `Connection.ID` to the remote
server URL and to the client chain assigned to this URL mapping remote server URL to a `connectClient`. Notably, on
every Close it is deleted from the `clients` map, so eventually its context will be canceled and the corresponding
`grpc.ClientConn` will be closed.

Care is taken to make sure that each client chain results in one increment of the refcount on its creation, and one
decrement when it receives a Close. In this way, we can be sure that `clienturl.NewClient(...)` context is closed after
the last client chain using it has received its Close.

The overall result is that usually, `connectServer` will have no more than one `clienturl.NewClient(...)` per `clientURL`.
It may occasionally have more than one in a transient fashion for the lifetime of one or more Connections.
In all events it will have zero `clienturl.NewClient(...)` for a `clientURL` if it has no server Connections for that
`clientURL`.

## connectClient

`connectClient` handles the instantiation and management of the client connection from the `clienturl.ClientURL(ctx)` of
the server Request.

## Monitoring

Both `connectServer` and `connectClient` by themselves doesn't monitor gRPC connection for liveness. Every error received
from Request, Close is simply returned to the previous chain elements. But there are some cases when connection should
be closed and reopened on some event happens. For this purpose `connectServer` injects into the client chain context a
cancel function for this context ([cancelctx](../../../tools/cancelctx/context.go)). When this context becomes canceled,
`connectClient` closes corresponding gRPC connection and `connectServer` force decreases its refcount to 0.

The most common way for the NSM chains is using [heal client](../heal/client.go) for monitoring and canceling the
connection, but it also can be implemented in some other way ([example using gRPC health check API](./monitor_client_test.go))
or not be implemented at all.
