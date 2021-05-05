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

// Package endpointtimeout provides server chain element that executes a callback when there were no connections for specified time
package endpointtimeout

import (
	"context"
	"os"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/clock"
)

type endpointTimeoutServer struct {
	ctx     context.Context
	timeout time.Duration
	notify  func()
	timer   clock.Timer
}

// NewServer - returns a new server chain element that notifies about long time periods without requests
func NewServer(ctx context.Context, options ...Option) networkservice.NetworkServiceServer {
	clockTime := clock.FromContext(ctx)

	t := &endpointTimeoutServer{
		ctx:     ctx,
		timeout: time.Minute * 10,
		notify:  func() { os.Exit(0) },
	}

	for _, opt := range options {
		opt(t)
	}

	t.timer = clockTime.Timer(t.timeout)

	go t.waitForTimeout()

	return t
}

func (t *endpointTimeoutServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if !t.timer.Stop() {
		return nil, errors.New("endpoint expired")
	}
	t.timer.Reset(t.timeout)

	return next.Server(ctx).Request(ctx, request)
}

func (t *endpointTimeoutServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}

func (t *endpointTimeoutServer) waitForTimeout() {
	select {
	case <-t.ctx.Done():
		return
	case <-t.timer.C():
		t.notify()
		return
	}
}
