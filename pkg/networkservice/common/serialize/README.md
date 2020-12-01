# Functional requirements

`Request`, `Close` events for the same `Connection.ID` should be executed in network service chain serially.

# Implementation

## serializeServer, serializeClient

Serialize chain elements use [multi executor](https://github.com/networkservicemesh/sdk/blob/master/pkg/tools/multiexecutor/multi_executor.go)
and stores per-id executor in the context.

Correct event firing chain element example:
```go
func (s *eventServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	executor := serialize.Executor(ctx)
	go func() {
		executor.AsyncExec(func() {
			_, _ = next.Server(ctx).Request(serialize.WithExecutor(context.TODO(), executor), request)
		})
	}()

	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		return nil, err
	}

	go func() {
		executor.AsyncExec(func() {
			_, _ = next.Server(ctx).Close(serialize.WithExecutor(context.TODO(), executor), conn)
		})
	}()

	return conn, nil
}
```
