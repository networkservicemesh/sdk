// Copyright (c) 2021-2024 Cisco and/or its affiliates.
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

package client

import (
	"net/url"
	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/begin"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/null"
)

type clientOptions struct {
	name                    string
	clientURL               *url.URL
	cc                      grpc.ClientConnInterface
	additionalFunctionality []networkservice.NetworkServiceClient
	authorizeClient         networkservice.NetworkServiceClient
	refreshClient           networkservice.NetworkServiceClient
	healClient              networkservice.NetworkServiceClient
	dialOptions             []grpc.DialOption
	dialTimeout             time.Duration
	reselectFunc            begin.ReselectFunc
}

// Option modifies default client chain values.
type Option func(c *clientOptions)

// WithName sets name for the client.
func WithName(name string) Option {
	return Option(func(c *clientOptions) {
		c.name = name
	})
}

// WithClientURL sets name for the client.
func WithClientURL(clientURL *url.URL) Option {
	return Option(func(c *clientOptions) {
		c.clientURL = clientURL
	})
}

// WithClientConn sets name for the client.
func WithClientConn(cc grpc.ClientConnInterface) Option {
	return Option(func(c *clientOptions) {
		c.cc = cc
	})
}

// WithAdditionalFunctionality sets additionalFunctionality for the client. Note: this adds into tail of the client chain.
func WithAdditionalFunctionality(additionalFunctionality ...networkservice.NetworkServiceClient) Option {
	return Option(func(c *clientOptions) {
		c.additionalFunctionality = additionalFunctionality
	})
}

// WithAuthorizeClient sets authorizeClient for the client chain.
func WithAuthorizeClient(authorizeClient networkservice.NetworkServiceClient) Option {
	if authorizeClient == nil {
		panic("authorizeClient cannot be nil")
	}
	return Option(func(c *clientOptions) {
		c.authorizeClient = authorizeClient
	})
}

// WithHealClient sets healClient for the client chain.
func WithHealClient(healClient networkservice.NetworkServiceClient) Option {
	if healClient == nil {
		panic("healClient cannot be nil")
	}
	return Option(func(c *clientOptions) {
		c.healClient = healClient
	})
}

// WithDialOptions sets dial options
func WithDialOptions(dialOptions ...grpc.DialOption) Option {
	return Option(func(c *clientOptions) {
		c.dialOptions = dialOptions
	})
}

// WithDialTimeout sets dial timeout
func WithDialTimeout(dialTimeout time.Duration) Option {
	return func(c *clientOptions) {
		c.dialTimeout = dialTimeout
	}
}

// WithoutRefresh disables refresh
func WithoutRefresh() Option {
	return func(c *clientOptions) {
		c.refreshClient = null.NewClient()
	}
}

// WithReselectFunc sets a function for changing request parameters on reselect
func WithReselectFunc(f func(*networkservice.NetworkServiceRequest)) Option {
	return func(c *clientOptions) {
		c.reselectFunc = f
	}
}
