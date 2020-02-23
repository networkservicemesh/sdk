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

// Package injecterror provides networkservice chain elements that simply returns an error.  Useful for testing.
package injecterror

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

type injectErrorClient struct {
	err *error
}

// NewClient - returns a testerror client that always returns an error when calling Request/Close
func NewClient(errs ...error) networkservice.NetworkServiceClient {
	var err *error
	if len(errs) > 0 {
		err = &(errs[0])
	}
	return &injectErrorClient{
		err: err,
	}
}

func (e *injectErrorClient) Request(ctx context.Context, in *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	if e.err != nil {
		return nil, *(e.err)
	}
	return nil, errors.New("error originates in injectErrorClient")
}

func (e *injectErrorClient) Close(ctx context.Context, in *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	if e.err != nil {
		return &empty.Empty{}, *(e.err)
	}
	return &empty.Empty{}, errors.New("error originates in injectErrorClient")
}
