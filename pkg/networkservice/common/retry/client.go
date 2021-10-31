// Copyright (c) 2021 Cisco and/or its affiliates.
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

// Package retry provides a networkservice.NetworksrviceClient wrapper that allows to retries requests and closes.
package retry

import (
	"context"
	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/networkservicemesh/sdk/pkg/tools/clock"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

// Settings represents retry policy settings.
type Settings struct {
	Timeout    time.Duration
	Interval   time.Duration
	TryTimeout time.Duration
}

type retryClient struct {
	Settings
	client networkservice.NetworkServiceClient
}

// Option configuress retry.Client instance.
type Option func(*retryClient)

// WithSettings sets retry policy settings for the retry.Client instance.
func WithSettings(s Settings) Option {
	return func(rc *retryClient) {
		rc.Settings = s
	}
}

// NewClient - returns a connect chain element
func NewClient(client networkservice.NetworkServiceClient, opts ...Option) networkservice.NetworkServiceClient {
	var result = &retryClient{
		Settings: Settings{
			Interval:   time.Millisecond * 200,
			TryTimeout: time.Second * 15,
			Timeout:    (time.Second*15 + time.Millisecond*200) * 5,
		},
		client: client,
	}

	for _, opt := range opts {
		opt(result)
	}

	return result
}

func (r *retryClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	logger := log.FromContext(ctx).WithField("retryClient", "Request")
	c := clock.FromContext(ctx)

	if r.Timeout > 0 {
		var cancel func()
		ctx, cancel = c.WithTimeout(ctx, r.Timeout)
		defer cancel()
	}

	for ctx.Err() == nil {
		requestCtx, cancel := c.WithTimeout(ctx, r.TryTimeout)
		resp, err := r.client.Request(requestCtx, request, opts...)
		cancel()

		if err != nil {
			logger.Errorf("try attempt has failed: %v", err.Error())

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-c.After(r.Interval):
				continue
			}
		}

		return resp, err
	}

	return nil, ctx.Err()
}

func (r *retryClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	logger := log.FromContext(ctx).WithField("retryClient", "Close")
	c := clock.FromContext(ctx)

	if r.Timeout > 0 {
		var cancel func()
		ctx, cancel = c.WithTimeout(ctx, r.Timeout)
		defer cancel()
	}

	for ctx.Err() == nil {
		closeCtx, cancel := c.WithTimeout(ctx, r.TryTimeout)

		resp, err := r.client.Close(closeCtx, conn, opts...)
		cancel()

		if err != nil {
			logger.Errorf("try attempt has failed: %v", err.Error())

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-c.After(r.Interval):
				continue
			}
		}

		return resp, err
	}

	return nil, ctx.Err()
}
