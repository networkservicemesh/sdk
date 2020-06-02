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

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/tools/spanhelper"
	"github.com/networkservicemesh/sdk/pkg/tools/typeutils"
)

type traceRegistryServer struct {
	traced registry.NetworkServiceRegistryServer
}

// NewNetworkServiceRegistryServer - wraps the provided registry.NetworkServiceRegistryServer in tracing
func NewNetworkServiceRegistryServer(traced registry.NetworkServiceRegistryServer) registry.NetworkServiceRegistryServer {
	return &traceRegistryServer{traced: traced}
}

func (t *traceRegistryServer) RegisterNSE(ctx context.Context, request *registry.NSERegistration) (*registry.NSERegistration, error) {
	// Create a new span
	operation := typeutils.GetFuncName(t.traced.RegisterNSE)
	span := spanhelper.FromContext(ctx, operation)
	defer span.Finish()

	// Make sure we log to span

	ctx = withLog(span.Context(), span.Logger())

	span.LogObject("request", request)

	// Actually call the next
	rv, err := t.traced.RegisterNSE(ctx, request)
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

type traceServer struct {
	registry.NetworkServiceRegistry_BulkRegisterNSEServer
	span   spanhelper.SpanHelper
	server registry.NetworkServiceRegistry_BulkRegisterNSEServer
}

func (ts *traceServer) Send(reg *registry.NSERegistration) error {
	ts.span.LogObject("send", reg)
	err := ts.server.Send(reg)
	ts.span.LogError(err)
	return err
}
func (ts *traceServer) Recv() (*registry.NSERegistration, error) {
	reg, err := ts.server.Recv()
	ts.span.LogError(err)
	ts.span.LogObject("recv", reg)
	return reg, err
}

func (ts *traceServer) Context() context.Context {
	return ts.span.Context()
}

// BulkRegisterNSE - register NSEs in a Bulk
func (t *traceRegistryServer) BulkRegisterNSE(server registry.NetworkServiceRegistry_BulkRegisterNSEServer) error {
	span := spanhelper.FromContext(server.Context(), typeutils.GetFuncName(t.traced.BulkRegisterNSE))
	span.Finish()

	stream := &traceServer{
		span:   span,
		server: server,
	}
	return t.traced.BulkRegisterNSE(stream)
}

func (t *traceRegistryServer) RemoveNSE(ctx context.Context, request *registry.RemoveNSERequest) (*empty.Empty, error) {
	// Create a new span
	operation := typeutils.GetFuncName(t.traced.RemoveNSE)
	span := spanhelper.FromContext(ctx, operation)
	defer span.Finish()

	// Make sure we log to span

	ctx = withLog(span.Context(), span.Logger())

	span.LogObject("request", request)

	// Actually call the next
	rv, err := t.traced.RemoveNSE(ctx, request)
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
