// Copyright (c) 2021 Doc.ai and/or its affiliates.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sandbox

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type restartableServer struct {
	ctx, currentCtx context.Context
	cancelCurrent   context.CancelFunc

	startFunction  func(context.Context)
	cancelFunction func()
}

func newRestartableServer(
	ctx context.Context,
	t *testing.T,
	serveURL *url.URL,
	startFunction func(ctx context.Context),
	cancelFunction func(),
) *restartableServer {
	cancelFunc := func() {}
	if cancelFunction != nil {
		cancelFunc = cancelFunction
	}
	r := &restartableServer{
		ctx:            ctx,
		cancelCurrent:  func() {},
		cancelFunction: cancelFunc,
		startFunction: func(ctx context.Context) {
			if !CheckURLFree(serveURL) {
				var timeout time.Duration
				if deadline, ok := ctx.Deadline(); ok {
					timeout = time.Until(deadline)
				} else {
					timeout = time.Second
				}

				require.Eventually(t, func() bool {
					return CheckURLFree(serveURL)
				}, timeout, timeout/100)
			}
			startFunction(ctx)
		},
	}

	r.Restart()

	return r
}

func (r *restartableServer) Restart() {
	r.Cancel()

	r.currentCtx, r.cancelCurrent = context.WithCancel(r.ctx)
	r.startFunction(r.currentCtx)
}

func (r *restartableServer) Cancel() {
	r.cancelFunction()
	r.cancelCurrent()
}
