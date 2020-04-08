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
	"fmt"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/tools/spanhelper"
	"github.com/networkservicemesh/sdk/pkg/tools/typeutils"
)

type traceRegistryClient struct {
	traced registry.NetworkServiceRegistryClient
}

// NewRegistryClient - wraps registry.NetworkServiceRegistryClient with tracing
func NewRegistryClient(traced registry.NetworkServiceRegistryClient) registry.NetworkServiceRegistryClient {
	return &traceRegistryClient{traced: traced}
}

func (t *traceRegistryClient) RegisterNSE(ctx context.Context, request *registry.NSERegistration, opts ...grpc.CallOption) (*registry.NSERegistration, error) {
	// Create a new span
	operation := fmt.Sprintf("%s/%s.RegisterNSE", typeutils.GetPkgPath(t.traced), typeutils.GetTypeName(t.traced))
	span := spanhelper.FromContext(ctx, operation)
	defer span.Finish()

	// Make sure we log to span

	ctx = withLog(span.Context(), span.Logger())

	span.LogObject("request", request)

	// Actually call the next
	rv, err := t.traced.RegisterNSE(ctx, request, opts...)
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

type traceClient struct {
	registry.NetworkServiceRegistry_BulkRegisterNSEClient
	span   spanhelper.SpanHelper
	client registry.NetworkServiceRegistry_BulkRegisterNSEClient
}

func (ts *traceClient) Send(reg *registry.NSERegistration) error {
	ts.span.LogObject("send", reg)
	err := ts.client.Send(reg)
	ts.span.LogError(err)
	return err
}
func (ts *traceClient) Recv() (*registry.NSERegistration, error) {
	reg, err := ts.client.Recv()
	ts.span.LogError(err)
	ts.span.LogObject("recv", reg)
	return reg, err
}

func (t *traceRegistryClient) BulkRegisterNSE(ctx context.Context, opts ...grpc.CallOption) (registry.NetworkServiceRegistry_BulkRegisterNSEClient, error) {
	// Create a new span
	span := spanhelper.FromContext(ctx, fmt.Sprintf("%s.BulkRegisterNSE", typeutils.GetTypeName(t.traced)))
	span.Finish() // Since events will be bound to span in any case
	ctx = withLog(span.Context(), span.Logger())

	cl, err := t.traced.BulkRegisterNSE(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return &traceClient{
		client: cl,
		span:   span,
	}, nil
}

func (t *traceRegistryClient) RemoveNSE(ctx context.Context, request *registry.RemoveNSERequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	// Create a new span
	span := spanhelper.FromContext(ctx, fmt.Sprintf("%s.Request", typeutils.GetTypeName(t.traced)))
	defer span.Finish()

	// Make sure we log to span

	ctx = withLog(span.Context(), span.Logger())

	span.LogObject("request", request)

	// Actually call the next
	rv, err := t.traced.RemoveNSE(ctx, request, opts...)
	if err != nil {
		span.LogError(err)
		return nil, err
	}
	span.LogObject("response", rv)
	return rv, err
}
