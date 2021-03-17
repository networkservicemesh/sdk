# Functional requirements

For some entities we need "expire on timeout if not being updated" logic. There are few requirements for the algorithm
we need to match:
1. Update can take some time and can fail:
   1. Expiration should be paused for the Update processing.
   2. If Update fails, expiration should be resumed for the same expiration time.
2. If some error occurs after the Update has been successfully finished, entity should be gracefully closed.
3. Close event should be performed on context with event scope lifetime, to prevent leaks.
4. If entity has been already closed or expired it should never be closed or expired until it will have been updated.

# Implementation

## Manager

`Manager` can be used for managing expiration for some set of entities. Here is an example for its usage with some
abstract `expireServer`:

```go
type expireServer struct {
	expireManager Manager
}

func (s *expireServer) Open(ctx context.Context, req *request) (*response, error) {
	logger := log.FromContext(ctx).WithField("expireServer", "Open")

	// 1. Stop expiration.
	s.expireManager.Stop(req.Id)

	// 2. Send Open event.
	resp, err := nextServer(ctx).Open(ctx, req)
	if err != nil {
		// 2.1. Reset expiration if Open event has failed.
		s.expireManager.Reset(req.Id)
		return nil, err
	}

	// 3. Delete the old expiration if we need to create a new one for the new ID.
	closeResp := resp.Clone()
	if closeResp.Id != req.Id {
		s.expireManager.Delete(req.Id)
	}

	// 4. Create a new expiration.
	s.expireManager.New(
		serializectx.GetExecutor(ctx, closeResp.Id),
		closeResp.Id,
		s.computeExpirationTime(closeResp),
		func (closeCtx context.Context) {
			if err := nextServer(ctx).Close(closeCtx, closeResp); err != nil {
				logger.Errorf("failed to close expired response: %s %s", closeResp.Id, err.Error())
			}
		},
	)

	return resp, nil
}

func (s *expireServer) Close(ctx context.Context, resp *response) error {
	logger := log.FromContext(ctx).WithField("expireServer", "Close")

	// 1. Check if we have an expiration.
	if !s.expireManager.DeleteExpiration(conn.GetId()) {
		// 1.1. If there is no expiration, there is nothing to do.
		logger.Warnf("response has been already closed: %s", resp.Id)
		return nil
	}

	// 2. Send Close event.
	return nextServer(ctx).Close(ctx, resp)
}
```