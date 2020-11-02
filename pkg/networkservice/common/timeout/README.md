# Functional requirements

If authentication token of the previous connection path element expires, it does mean that we lost connection with the
previous NSM-based application. For such case connection should be closed.

# Implementation

## timeoutServer

timeoutServer keeps timers [timeout.timerMap](https://github.com/networkservicemesh/sdk/blob/master/pkg/networkservice/common/timeout/gen.go#L26)
mapping incoming request Connection.ID to a timeout timer firing Close on the subsequent chain after the connection previous
path element expires. To prevent simultaneous execution of multiple Request, Close event for the same Connection.ID in parallel
it also keeps executors [timeout.executorMap](https://github.com/networkservicemesh/sdk/blob/master/pkg/networkservice/common/timeout/gen.go#L27)
mapping request Connection.ID to an executor for serializing all Request, Close event for the mapped Connection.ID.

timeoutServer closes only subsequent chain elements and uses base context for the Close. So all the chain elements in
the chain before the timeoutServer shouldn't be closed with Close and shouldn't set any required data to the Close context.

Example of correct timeoutServer usage in [endpoint.NewServer](https://github.com/networkservicemesh/sdk/blob/master/pkg/networkservice/chains/endpoint/server.go#L62):
```go
rv.NetworkServiceServer = chain.NewNetworkServiceServer(
    append([]networkservice.NetworkServiceServer{
        authzServer, // <-- shouldn't be closed, don't set anything to the context
        updatepath.NewServer(name), // <-- same
        timeout.NewServer(ctx), // <-- timeoutServer
        monitor.NewServer(ctx, &rv.MonitorConnectionServer), // <-- should be closed
        updatetoken.NewServer(tokenGenerator), // <-- should be closed
    }, additionalFunctionality...)...) // <-- should be closed, sets data to context
```

## Comments on concurrency characteristics

Concurrency is managed through type specific wrappers of [sync.Map](https://golang.org/pkg/sync/#Map) and with
per-connection [serialize.Executor](https://github.com/edwarnicke/serialize/blob/master/serialize.go) which are created on
Request and deleted on Close.

Since we are deleting the per-connection executor on connection Close, there possibly can be a race condition:
```
1. -> timeout close  : locking executor
2. -> request-1      : waiting on executor
3. timeout close ->  : unlocking executor, removing it from executors
4. -> request-2      : creating exec, storing into executors, locking exec
5. -request-1->      : locking executor, trying to store it into executors
```
at 5. we get request-1 locking executor, request-2 locking exec and only exec stored in executors. It means that
request-2 and all subsequent events will be executed in parallel with request-1.
So we check for this [here](https://github.com/networkservicemesh/sdk/blob/master/pkg/networkservice/common/timeout/server.go#L68).