# Functional requirements

Upon receipt of a Request, the connect Server chain element must send, as a client, a corresponding request to 
another Server.  For clarity, we will refer to the incoming Request to the server as the 'server Request'.

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

## connectClient
Each incoming server Connection must be translated to its corresponding client Connection for subsequent server Request and Server Close
calls. For this reason, connectClient is implemented, to manage that translation.  This results in one 'client' per server Connection.
Each connectClient processes maximally one Request/Close at a time, enforced with an executor.  The connectClient performs
the proper translation (or storage) of the client Connection and then calls the next.Client(ctx).  Upon receiving a Close, connectClient,
calls a 'cancel' function provided to it at its construction.

## connectServer

connectServer keeps clientsByID [clientmap.Map](https://github.com/networkservicemesh/sdk/blob/master/pkg/tools/clientmap/gen.go#L27) mapping incoming server Connection.ID
to a chain consisting of the corresponding connectClient and a 
[clienturl.NewClient(...)](https://github.com/networkservicemesh/sdk/blob/master/pkg/networkservice/common/clienturl/client.go#L44)
which handles the instantiation and management of the client connection from the clienturl.ClientURL(ctx) of the server 
Request.  Notably, 
[clienturl.NewClient(...)](https://github.com/networkservicemesh/sdk/blob/master/pkg/networkservice/common/clienturl/client.go#L44)
uses [setFinalizer](https://github.com/networkservicemesh/sdk/blob/master/pkg/networkservice/common/clienturl/client.go#L80) to ensure
that once it is garbage collected its context is cancelled and its corresponding grpc.ClientConn is Closed.

connectServer also keeps a clientsByURL
[clientmap.RefcountMap](https://github.com/networkservicemesh/sdk/blob/master/pkg/tools/clientmap/refcount.go#L43) of 
[clienturl.NewClient(...)](https://github.com/networkservicemesh/sdk/blob/master/pkg/networkservice/common/clienturl/client.go#L44).

Care is taken to make sure that each connectClient results in one increment of the refcount on its creation, and one decrement when
it receives a Close.  In this way, we can be sure that:

1. clienturl.NewClient(...) is not referenced in clientsByURL after the last connectClient using it has received its Close.
2. If for any reason the clienturl.NewClient(...) is deleted prematurely from the clientsByURL map, [which can happen](https://github.com/networkservicemesh/sdk/blob/master/pkg/tools/clientmap/refcount.go#L78),
any connectClients actively using it retain their pointer to it, and can continue to utilize it throughout their lifetime.

The overall result is that usually, connectServer will have no more than one clienturl.NewClient(...) per clientURL.
It may occasionally have more than one in a transient fashion for the lifetime of one or more Connections.  In all events
it will have zero clienturl.NewClient(...)s for a clientURL if it has no server Connections for that clientURL.

## Comments on concurrency characteristics.

Concurrency is primarily managed through type-specific wrappers of [sync.Map](https://golang.org/pkg/sync/#Map):
> The Map type is optimized for two common use cases: (1) when the entry for a given key is only ever written once 
> but read many times, as in caches that only grow, or (2) when multiple goroutines read, write, and overwrite entries 
> for disjoint sets of keys. In these two cases, use of a Map may significantly reduce lock contention compared to a 
> Go map paired with a separate Mutex or RWMutex.

This is precisely our case.

[clientmap.RefcountMap](https://github.com/networkservicemesh/sdk/blob/master/pkg/tools/clientmap/refcount.go#L43) utilizes
a simple atomic refcount, which is also highly performant.

connectClient itself is fully serialized by design.  It should almost never happen that we receive more than one Request/Close
before the one before it can be processed, so this will almost never have appreciable performance impact.  In the unusual 
circumstance we do, it will retain correctness via its internal serialize.Executor.
