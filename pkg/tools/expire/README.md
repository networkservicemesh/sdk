# Functional requirements

For some entities we need "expire on timeout if not being updated" logic. There are few requirements for the algorithm
we need to match:
1. Update can take some time and can fail:
   1. Expiration should be paused for the Update processing.
   2. If Update fails, expiration should be resumed for the same expiration time.
2. If some error occurs after the Update has been successfully finished, entity should be gracefully closed.
3. Close event should be performed on context with event scope lifetime, to prevent leaks.
4. If entity has been already closed or expired it should never be closed or expired until it will have been updated.

# Abstract implementation

```go
// WARNING: `expireServer` uses ctx as a context for the Close event, so if there are any chain elements setting some
// data in context in chain before the `expireServer`, these changes won't appear in the Close context.
type expireServer struct {
	ctx    context.Context
	timers TimerMap
}

func (s *expireServer) Open(ctx context.Context, req *request) (*response, error) {
	clockTime := clock.FromContext(ctx)
	logger := log.FromContext(ctx).WithField("expireServer", "Open")

	// 1. Try to load and stop timer.
	t, loaded := s.timers.Load(req.Id)
	stopped := loaded && t.Stop()

	// 2. Send Open event.
	resp, err := nextServer(ctx).Open(ctx, req)
	if err != nil {
		if stopped {
			// 2.1. Reset timer if Open event has failed and timer has been successfully stopped.
			t.Reset(clockTime.Until(t.ExpirationTime))
		}
		return nil, err
	}

	// 3. Delete the old timer
	s.timers.Delete(req.Id)

	// 4. Create a new timer.
	t, err = s.newTimer(ctx, resp.Clone())
	if err != nil {
		// 4.1. If we have failed to create a new timer, Close the response.
		if closeErr := nextServer(ctx).Close(ctx, resp); closeErr != nil {
			logger.Errorf("failed to close response on error: %s %s", resp.Id, closeErr.Error())
		}
		return nil, err
	}

	// 5. Store timer.
	s.timers.Store(resp.Id, t)

	return resp, nil
}

func (s *expireServer) newTimer(ctx context.Context, resp *response) (*Timer, error) {
	clockTime := clock.FromContext(ctx)
	logger := log.FromContext(ctx).WithField("expireServer", "newTimer")

	// 1. Try to get executor from the context.
	executor := serializectx.GetExecutor(ctx, resp.Id)
	if executor == nil {
		// 1.1. Fail if there is no executor.
		return nil, errors.Errorf("failed to get executor from context")
	}

	// 2. Compute expiration time.
	expirationTime := ...

	// 3. Create timer.
	var t *Timer
	t = &Timer{
		ExpirationTime: expirationTime,
		Timer: clockTime.AfterFunc(clockTime.Until(expirationTime), func() {
			// 3.1. All the timer action should be executed under the `executor.AsyncExec`.
			executor.AsyncExec(func() {
				// 3.2. Timer has probably been stopped and deleted or replaced with a new one, so we need to check it
				// before deleting.
				if tt, ok := s.timers.Load(resp.Id); !ok || tt != t {
					// 3.2.1. This timer has been stopped, nothing to do.
					return
				}

				// 3.3. Delete timer.
				s.timers.Delete(resp.Id)

				// 3.4. Since `s.ctx` lives with the application, we need to create a new context with event scope
				// lifetime to prevent leaks.
				closeCtx, cancel := context.WithCancel(s.ctx)
				defer cancel()

				// 3.5. Close expired response.
				if err := nextServer(ctx).Close(closeCtx, resp); err != nil {
					logger.Errorf("failed to close expired response: %s %s", resp.Id, err.Error())
				}
			})
		}),
	}

	return t, nil
}

func (s *expireServer) Close(ctx context.Context, resp *response) error {
	logger := log.FromContext(ctx).WithField("expireServer", "Close")

	// 1. Check if we have a timer.
	t, ok := s.timers.LoadAndDelete(resp.Id)
	if !ok {
		// 1.1. If there is no timer, there is nothing to do.
		logger.Warnf("response has been already closed: %s", resp.Id)
		return nil
	}

	// 2. Stop it.
	t.Stop()

	// 3. Send Close event.
	return nextServer(ctx).Close(ctx, resp)
}
```