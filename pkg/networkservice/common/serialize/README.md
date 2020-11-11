# Functional requirements

`Request`, `Close` events for the same `Connection.ID` should be executed in network service chain serially.

# Implementation

## serializeServer, serializeClient

serialize chain elements keep [serialize.executorMap](https://github.com/networkservicemesh/sdk/blob/master/pkg/networkservice/common/serialize/gen.go#L25)
mapping incoming `Connection.ID` to a [serialize.Executor](https://github.com/edwarnicke/serialize/blob/master/serialize.go).
New executors are created on Request. Close deletes existing executor for the request.

To make possible a new chain element firing asynchronously with `Request`, `Close` events, serialize chain elements wraps
per-connection executor with [serialize.CancellableExecutor](https://github.com/networkservicemesh/sdk/blob/master/pkg/networkservice/common/serialize/cancellable_executor.go)
and inserts it into the `Request` context. Such thing is not performed for the `Close` context because the executor will
already be cancelled by the time it becomes free.

Correct `Request` firing chain element example:
```go
func (s *requestServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	executor := serialize.Executor(ctx)
	go func() {
		executor.AsyncExec(func() {
			_, _ = next.Server(ctx).Request(serialize.WithExecutor(context.TODO(), executor), request)
		})
	}()

	return next.Server(ctx).Request(ctx, request)
}
```

Correct `Close` firing chain element example:
```go
func (s *closeServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		return nil, err
	}

	executor := serialize.Executor(ctx)
	go func() {
		executor.AsyncExec(func() {
			_, _ = next.Server(ctx).Close(context.TODO(), conn)
			executor.Cancel()
		})
	}()

	return conn, err
}
```

# race condition case

There is a possible case for the race condition:
```
     1. -> close      : locking `executor`
     2. -> request-1  : waiting on `executor`
     3. close ->      : unlocking `executor`, removing it from `executors`
     4. -> request-2  : creating `exec`, storing into `executors`, locking `exec`
     5. -request-1->  : locking `executor`, trying to store it into `executors`
```
at 5. we get `request-1` locking `executor`, `request-2` locking `exec` and only `exec` stored in `executors`. It means
that `request-2` and all subsequent events will be executed in parallel with `request-1`. If such case happens, `request-1`
fails with _race condition_ error.