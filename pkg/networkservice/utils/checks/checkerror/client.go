// Copyright (c) 2020 Cisco and/or its affiliates.
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

package checkerror

import (
	"context"
	"testing"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type checkErrorClient struct {
	*testing.T
	isNil bool
	err   *error
}

// NewClient - returns NetworkServiceClient chain element that checks for an error being returned from the next element in the chain
//             t - *testing.T for checking
//             isNil - if true, check that error is nil, if false, check that err is not nil.
//             errs - optional - if errs[0] is provided, will check to see if the error matches errs[0]
func NewClient(t *testing.T, isNil bool, errs ...error) networkservice.NetworkServiceClient {
	rv := &checkErrorClient{
		T:     t,
		isNil: isNil,
	}
	if len(errs) > 0 {
		rv.err = &(errs[0])
	}
	return rv
}

func (c *checkErrorClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if c.isNil {
		assert.Nil(c.T, err)
		return conn, err
	}
	assert.NotNil(c.T, err)
	if c.err != nil {
		assert.Equal(c.T, &err, c.err)
	}
	return conn, err
}

func (c *checkErrorClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	e, err := next.Client(ctx).Close(ctx, conn, opts...)
	if c.isNil {
		assert.Nil(c.T, err)
		return e, err
	}
	assert.NotNil(c.T, err)
	if c.err != nil {
		assert.Equal(c.T, &err, c.err)
	}
	return e, err
}
