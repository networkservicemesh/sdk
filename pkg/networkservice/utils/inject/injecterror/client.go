// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
//
// Copyright (c) 2020-2021 Cisco and/or its affiliates.
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

package injecterror

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type injectErrorClient struct {
	requestErrorSupplier, closeErrorSupplier *errorSupplier
}

// NewClient returns a client chain element returning error on Close/Request on given times.
func NewClient(opts ...Option) networkservice.NetworkServiceClient {
	o := &options{
		err:               errors.New("error originates in injectErrorClient"),
		requestErrorTimes: []int{-1},
		closeErrorTimes:   []int{-1},
	}

	for _, opt := range opts {
		opt(o)
	}

	return &injectErrorClient{
		requestErrorSupplier: &errorSupplier{
			err:        o.err,
			errorTimes: o.requestErrorTimes,
		},
		closeErrorSupplier: &errorSupplier{
			err:        o.err,
			errorTimes: o.closeErrorTimes,
		},
	}
}

func (c *injectErrorClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	if err := c.requestErrorSupplier.supply(); err != nil {
		return nil, err
	}
	return next.Client(ctx).Request(ctx, request, opts...)
}

func (c *injectErrorClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	if err := c.closeErrorSupplier.supply(); err != nil {
		return nil, err
	}
	return next.Client(ctx).Close(ctx, conn, opts...)
}
