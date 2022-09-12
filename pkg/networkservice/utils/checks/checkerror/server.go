// Copyright (c) 2020-2022 Cisco and/or its affiliates.
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

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type checkErrorServer struct {
	*testing.T
	isNil bool
	err   *error
}

// NewServer - returns NetworkServiceServer chain element that checks for an error being returned from the next element in the chain
//
//	t - *testing.T for checking
//	isNil - if true, check that error is nil, if false, check that err is not nil.
//	errs - optional - if errs[0] is provided, will check to see if the error matches errs[0]
func NewServer(t *testing.T, isNil bool, errs ...error) networkservice.NetworkServiceServer {
	rv := &checkErrorServer{
		T:     t,
		isNil: isNil,
	}
	if len(errs) > 0 {
		rv.err = &(errs[0])
	}
	return rv
}

func (c *checkErrorServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	conn, err := next.Server(ctx).Request(ctx, request)
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

func (c *checkErrorServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	e, err := next.Server(ctx).Close(ctx, conn)
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
