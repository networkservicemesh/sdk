# Functional requirements

`Request`, `Close` events for the same `Connection.ID` should be executed in network service chain serially.

# Implementation

## serializeServer, serializeClient

Serialize chain elements uses [multi executor](https://github.com/networkservicemesh/sdk/blob/master/pkg/tools/multiexecutor/multi_executor.go)
and stores per-id executor in the `Request` context.\
**NOTE:** we don't pass executor to the `Close` context because it is very strange to plan events on already closed
connection. Please don't do it.

Correct event firing chain element example:
```go
func (s *eventServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	executor := serializectx.Executor(ctx, request.GetConnection().GetId())
	go func() {
		executor.AsyncExec(func() {
			_, _ = next.Server(ctx).Request(serializectx.WithExecutor(context.TODO(), executor), request)
		})
	}()

	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		return nil, err
	}

	go func() {
		executor.AsyncExec(func() {
			// We don't pass executor to the Close context. Please don't do it.
			_, _ = next.Server(ctx).Close(context.TODO(), conn)
		})
	}()

	return conn, nil
}
```
