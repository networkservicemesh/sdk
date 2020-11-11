# Functional requirements

If authentication token of the previous connection path element expires, it does mean that we lost connection with the
previous NSM-based application. For such case connection should be closed.

# Implementation

## timeoutServer

timeoutServer keeps timers [timeout.timerMap](https://github.com/networkservicemesh/sdk/blob/master/pkg/networkservice/common/timeout/gen.go#L26)
mapping incoming request Connection.ID to a timeout timer firing Close on the subsequent chain after the connection previous
path element expires.

timeoutServer closes only subsequent chain elements and uses base context for the Close. So all the chain elements in
the chain before the timeoutServer shouldn't be closed with Close and shouldn't set any required data to the Close context.

Example of correct timeoutServer usage in [endpoint.NewServer](https://github.com/networkservicemesh/sdk/blob/master/pkg/networkservice/chains/endpoint/server.go#L62):
```go
rv.NetworkServiceServer = chain.NewNetworkServiceServer(
    append([]networkservice.NetworkServiceServer{
        authzServer, // <-- shouldn't be closed, don't set anything to the context
        updatepath.NewServer(name), // <-- same
        serialize.NewServer(ctx) // <-- should be before the timeoutServer
        timeout.NewServer(ctx), // <-- timeoutServer
        monitor.NewServer(ctx, &rv.MonitorConnectionServer), // <-- should be closed
        updatetoken.NewServer(tokenGenerator), // <-- should be closed
    }, additionalFunctionality...)...) // <-- should be closed, sets data to context
```

## Comments on concurrency characteristics

Concurrency is managed with [serialize.NewServer](https://github.com/networkservicemesh/sdk/blob/master/pkg/networkservice/common/serialize/server.go)
in the chain before.