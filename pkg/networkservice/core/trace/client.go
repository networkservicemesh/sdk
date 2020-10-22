// Copyright (c) 2020 Cisco Systems, Inc.
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
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/tools/spanhelper"
	"github.com/networkservicemesh/sdk/pkg/tools/typeutils"
)

type traceClient struct {
	traced networkservice.NetworkServiceClient
}

// NewNetworkServiceClient - wraps tracing around the supplied networkservice.NetworkServiceClient
func NewNetworkServiceClient(traced networkservice.NetworkServiceClient) networkservice.NetworkServiceClient {
	return &traceClient{traced: traced}
}

func (t *traceClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	// Create a new span
	operation := typeutils.GetFuncName(t.traced, "Request")
	span := spanhelper.FromContext(ctx, operation)
	defer span.Finish()

	ctx = withLog(span.Context(), span.Logger())
	logRequest(span, request)

	// Actually call the next
	rv, err := t.traced.Request(ctx, request, opts...)

	if err != nil {
		if _, ok := err.(stackTracer); !ok {
			err = errors.Wrapf(err, "Error returned from %s", operation)
			span.LogErrorf("%+v", err)
			return nil, err
		}
		span.LogErrorf("%v", err)
		return nil, err
	}

	logResponse(span, rv)
	return rv, err
}

func (t *traceClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	// Create a new span
	operation := typeutils.GetFuncName(t.traced, "Close")
	span := spanhelper.FromContext(ctx, operation)
	defer span.Finish()
	// Make sure we log to span
	ctx = withLog(span.Context(), span.Logger())

	logRequest(span, conn)
	rv, err := t.traced.Close(ctx, conn, opts...)

	if err != nil {
		if _, ok := err.(stackTracer); !ok {
			err = errors.Wrapf(err, "Error returned from %s", operation)
			span.LogErrorf("%+v", err)
			return nil, err
		}
		span.LogErrorf("%v", err)
		return nil, err
	}
	span.LogObject("response", rv)
	return rv, err
}
