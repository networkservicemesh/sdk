package heal

import (
	"context"
	"sync"
)

type ctxWrapper struct {
	cancel context.CancelFunc
	lock   sync.Mutex
}

type findCtxWrapper struct {
	ctx       context.Context
	restoreCh chan struct{}
	resultCh  chan error

	ctxWrapper
}
