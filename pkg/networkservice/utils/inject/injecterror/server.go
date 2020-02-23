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

package injecterror

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/pkg/errors"
)

type injectErrorServer struct {
	err *error
}

// NewServer - returns a testerror server that always returns an error when calling Request/Close
func NewServer(errs ...error) networkservice.NetworkServiceServer {
	var err *error
	if len(errs) > 0 {
		err = &errs[0]
	}
	return &injectErrorServer{
		err: err,
	}
}

func (e injectErrorServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if e.err != nil {
		return nil, *(e.err)
	}
	return nil, errors.New("error originated in injectErrorServer")
}

func (e injectErrorServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	if e.err != nil {
		return &empty.Empty{}, *(e.err)
	}
	return &empty.Empty{}, errors.New("error originated in injectErrorServer")
}
