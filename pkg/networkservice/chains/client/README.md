# Functional requirements

There are some common chain elements that we expect to have in every client chain to make NSM working. Instead of that,
there are few different scenarios when we need to create a client chain to initiate NSM request:

1. Client to NSMgr - simple case when there is an application requesting some L2/L3 connection from the NSMgr.
    * no incoming L2/L3 request - client itself is a request generator
    * complete chain
    ```
    Client   --Request-->   NSMgr
       |                      |
       |---L2/L3 connection---|
       |                      |
    ```
2. Server to endpoint client - we already have application running as a NSM endpoint receiving request to L2/L3
   connection, but it also needs to request some other L2/L3 connection from some other endpoint.
    * there is an incoming L2/L3 request - we need to generate an outgoing L2/L3 request, but the connection we return
      is an incoming connection
    * part of some server chain - we need to add `clientConnection` and request next elements
    * different endpoints mean different connections, even if we have same `Connection.Id` for them
    ```
    ...                 Endpoint  --Request-->  Endpoint
     |                      |                      |
     |---L2/L3 connection---|---L2/L3 connection---|
     |                      |                      |
    ```
3. Proxy to endpoint client - we already have application running as a NSM server, but it doesn't provide L2/L3
   connection, it simply passes the request to some other endpoint.
    * there is an incoming L2/L3 request but we simply forward it
    * part of some server chain - we need to add `clientConnection` and request next elements
    * different endpoints mean different connections, even if we have same `Connection.Id` for them
    ```
    ...                   Proxy   --Request-->  Endpoint
     |                      |                      |
     |---------------L2/L3 connection--------------|
     |                      |                      |
    ```

# Implementation

## client.NewClient(..., grpcCC, ...additionalFunctionality)

It is a solution for the (1.) case. Client appends `additionalFunctionality` to the default client chain and passes
incoming request to the NSMgr over the `grpcCC`.

## client.NewClientFactory(..., ...additionalFunctionalityGenerators)

It is a solution for the (2.), (3.) cases. We create a new GRPC client on each new client URL received from the incoming
request. It can be used in `connect.NewServer` so `clientConnection` will be processed correctly. If we need to
implement mechanism translation as for (3.), we can add some mechanism translation client generator as a part of
`additionalFunctionalityGenerators`.
