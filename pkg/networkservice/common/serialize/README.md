# Functional requirements

`Request`, `Close` events for the same `Connection.ID` should be executed in network service chain serially.

# Implementation

## serializer

`serializer` maps incoming `Connection.ID` to a [serialize.Executor](https://github.com/edwarnicke/serialize/blob/master/serialize.go),
request count and state. New mappings are created on Request. If request count and state are equal 0, mapping becomes
free and can be cleaned up. Request count is incremented with each Request before requesting the next chain element and
decrements after. All request count modifications are performed under the `serializer.executor`. State is being changed
in Request from 0 to random, or in Close from not 0 to 0. If state is equal 0, connection is closed so all incoming
Close events should not be processed.

To make possible a new chain element firing asynchronously with `Request`, `Close` events, serialize chain elements wraps
per-connection executor with [request executor](https://github.com/networkservicemesh/sdk/blob/master/pkg/networkservice/common/serialize/executor.go#L36),
[close executor](https://github.com/networkservicemesh/sdk/blob/master/pkg/networkservice/common/serialize/executor.go#L40)
and inserts them into the `Request` context. Such a thing is not being performed for the `Close` context because the
executor will already be cancelled by the time it becomes free. For the generated events state should be equal the
original state for the Request/CloseExecutor.

Correct `Request` firing chain element example:
```go
func (s *requestServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
    executor := serialize.RequestExecutor(ctx)
    go func() {
        executor.AsyncExec(func() error {
            _, _ = next.Server(ctx).Request(serialize.WithExecutorsFromContext(context.TODO(), ctx), request)
            return nil
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

    executor := serialize.CloseExecutor(ctx)
    go func() {
        executor.AsyncExec(func() error {
            _, _ = next.Server(ctx).Close(context.TODO(), conn)
            return errors.New() // I don't want to close the chain.
        })
    }()
    go func() {
        executor.AsyncExec(func() error {
            _, _ = next.Server(ctx).Close(context.TODO(), conn)
            return nil // I want to close the chain.
        })
    }()

    return conn, err
}
```
