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

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/tools/spanhelper"
	"github.com/networkservicemesh/sdk/pkg/tools/typeutils"
)

type traceRegistryServer struct {
	traced registry.NetworkServiceRegistryServer
}

// NewRegistryServer - wraps the provided registry.NetworkServiceRegistryServer in tracing
func NewRegistryServer(traced registry.NetworkServiceRegistryServer) registry.NetworkServiceRegistryServer {
	return &traceRegistryServer{traced: traced}
}

func (t *traceRegistryServer) RegisterNSE(ctx context.Context, request *registry.NSERegistration) (*registry.NSERegistration, error) {
	// Create a new span
	span := spanhelper.FromContext(ctx, fmt.Sprintf("%s.Request", typeutils.GetTypeName(t.traced)))
	defer span.Finish()

	// Make sure we log to span

	ctx = withLog(span.Context(), span.Logger())

	span.LogObject("request", request)

	// Actually call the next
	rv, err := t.traced.RegisterNSE(ctx, request)
	if err != nil {
		span.LogError(err)
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

// BulkRegisterNSE - register NSEs in a Bulk
func (t *traceRegistryServer) BulkRegisterNSE(server registry.NetworkServiceRegistry_BulkRegisterNSEServer) error {
	span := spanhelper.FromContext(server.Context(), fmt.Sprintf("%s.Request", typeutils.GetTypeName(t.traced)))
	span.Finish()

	stream := &traceServer{
		span:   span,
		server: server,
	}
	return t.traced.BulkRegisterNSE(stream)
}

func (t *traceRegistryServer) RemoveNSE(ctx context.Context, request *registry.RemoveNSERequest) (*empty.Empty, error) {
	// Create a new span
	span := spanhelper.FromContext(ctx, fmt.Sprintf("%s.Request", typeutils.GetTypeName(t.traced)))
	defer span.Finish()

	// Make sure we log to span

	ctx = withLog(span.Context(), span.Logger())

	span.LogObject("request", request)

	// Actually call the next
	rv, err := t.traced.RemoveNSE(ctx, request)
	if err != nil {
		span.LogError(err)
		return nil, err
	}
	span.LogObject("response", rv)
	return rv, err
}
