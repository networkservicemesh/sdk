// Copyright (c) 2020 Cisco Systems, Inc.
//
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

package trace

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/tools/logger"

	"github.com/networkservicemesh/sdk/pkg/tools/typeutils"
)

type traceServer struct {
	traced networkservice.NetworkServiceServer
}

// NewNetworkServiceServer - wraps tracing around the supplied traced
func NewNetworkServiceServer(traced networkservice.NetworkServiceServer) networkservice.NetworkServiceServer {
	return &traceServer{traced: traced}
}

func (t *traceServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	// Create a new logger
	operation := typeutils.GetFuncName(t.traced, "Request")
	ctx, finish := withLog(ctx, operation)
	defer finish()

	logRequest(ctx, request)

	// Actually call the next
	rv, err := t.traced.Request(ctx, request)

	if err != nil {
		if _, ok := err.(stackTracer); !ok {
			err = errors.Wrapf(err, "Error returned from %s", operation)
			logger.Log(ctx).Errorf("%+v", err)
			return nil, err
		}
		logger.Log(ctx).Errorf("%v", err)
		return nil, err
	}

	logResponse(ctx, rv)
	return rv, err
}

func (t *traceServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	// Create a new logger
	operation := typeutils.GetFuncName(t.traced, "Close")
	ctx, finish := withLog(ctx, operation)
	defer finish()

	logRequest(ctx, conn)
	rv, err := t.traced.Close(ctx, conn)

	if err != nil {
		if _, ok := err.(stackTracer); !ok {
			err = errors.Wrapf(err, "Error returned from %s", operation)
			logger.Log(ctx).Errorf("%+v", err)
			return nil, err
		}
		logger.Log(ctx).Errorf("%v", err)
		return nil, err
	}
	return rv, err
}
