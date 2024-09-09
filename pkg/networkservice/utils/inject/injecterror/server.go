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

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type injectErrorServer struct {
	requestErrorSupplier, closeErrorSupplier *errorSupplier
}

// NewServer returns a server chain element returning error on Close/Request on given times.
func NewServer(opts ...Option) networkservice.NetworkServiceServer {
	o := &options{
		err:               errors.New("error originates in injectErrorServer"),
		requestErrorTimes: []int{-1},
		closeErrorTimes:   []int{-1},
	}

	for _, opt := range opts {
		opt(o)
	}

	return &injectErrorServer{
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

func (s injectErrorServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if err := s.requestErrorSupplier.supply(); err != nil {
		return nil, err
	}
	return next.Server(ctx).Request(ctx, request)
}

func (s injectErrorServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	if err := s.closeErrorSupplier.supply(); err != nil {
		return nil, err
	}
	return next.Server(ctx).Close(ctx, conn)
}
