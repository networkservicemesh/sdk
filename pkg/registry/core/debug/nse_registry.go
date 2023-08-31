// Copyright (c) 2023 Doc.ai and/or its affiliates.
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

package debug

import (
	"context"
	"io"

	"github.com/networkservicemesh/sdk/pkg/registry/core/streamcontext"
	"github.com/networkservicemesh/sdk/pkg/tools/typeutils"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/networkservicemesh/api/pkg/api/registry"
)

type debugNetworkServiceEndpointRegistryClient struct {
	debugged registry.NetworkServiceEndpointRegistryClient
}

type debugNetworkServiceEndpointRegistryFindClient struct {
	registry.NetworkServiceEndpointRegistry_FindClient
}

const (
	nseClientRegisterLoggedKey   contextKeyType = "nseClientRegisterLoggedKey"
	nseClientFindLoggedKey       contextKeyType = "nseClientFindLoggedKey"
	nseClientUnregisterLoggedKey contextKeyType = "nseClientUnregisterLoggedKey"
	nseServerRegisterLoggedKey   contextKeyType = "nseServerRegisterLoggedKey"
	nseServerFindLoggedKey       contextKeyType = "nseServerFindLoggedKey"
	nseServerUnregisterLoggedKey contextKeyType = "nseServerUnregisterLoggedKey"
)

func (t *debugNetworkServiceEndpointRegistryFindClient) Recv() (*registry.NetworkServiceEndpointResponse, error) {
	operation := typeutils.GetFuncName(t.NetworkServiceEndpointRegistry_FindClient, methodNameRecv)
	updatedContext := withLog(t.Context())

	s := streamcontext.NetworkServiceEndpointRegistryFindClient(updatedContext, t.NetworkServiceEndpointRegistry_FindClient)
	rv, err := s.Recv()

	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, errors.WithStack(err)
		}
		if status.Code(err) == codes.Canceled {
			return nil, errors.WithStack(err)
		}

		if isReadyForLogging(updatedContext, true) && storeNSEClientRecvErrorLogged(updatedContext) {
			return nil, logError(updatedContext, err, operation)
		}

		return nil, err
	}

	if isReadyForLogging(updatedContext, true) && storeNSEClientRecvLogged(updatedContext) {
		logObjectDebug(updatedContext, "nse-recv-response", rv)
	}

	return rv, nil
}

func (t *debugNetworkServiceEndpointRegistryClient) Register(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	operation := typeutils.GetFuncName(t.debugged, methodNameRegister)
	updatedContext := withLog(ctx)

	if updatedContext.Value(nseClientRegisterLoggedKey) == nil {
		updatedContext = context.WithValue(updatedContext, nseClientRegisterLoggedKey, t)
		logObjectDebug(updatedContext, "nse-register", in)
	}

	rv, err := t.debugged.Register(updatedContext, in, opts...)
	if err != nil {
		if updatedContext.Value(nseClientRegisterLoggedKey) == t {
			return nil, logError(updatedContext, err, operation)
		}
		return nil, err
	}

	if updatedContext.Value(nseClientRegisterLoggedKey) == t {
		logObjectDebug(updatedContext, "nse-register-response", rv)
	}
	return rv, nil
}
func (t *debugNetworkServiceEndpointRegistryClient) Find(ctx context.Context, in *registry.NetworkServiceEndpointQuery, opts ...grpc.CallOption) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	operation := typeutils.GetFuncName(t.debugged, methodNameFind)
	updatedContext := withLog(ctx)

	if updatedContext.Value(nseClientFindLoggedKey) == nil {
		updatedContext = context.WithValue(updatedContext, nseClientFindLoggedKey, t)
		logObjectDebug(updatedContext, "nse-find", in)
	}

	// Actually call the next
	rv, err := t.debugged.Find(updatedContext, in, opts...)
	if err != nil {
		if updatedContext.Value(nseClientFindLoggedKey) == t {
			return nil, logError(updatedContext, err, operation)
		}
		return nil, err
	}

	if updatedContext.Value(nseClientFindLoggedKey) == t {
		logObjectDebug(updatedContext, "nse-find-response", rv)
	}
	return &debugNetworkServiceEndpointRegistryFindClient{NetworkServiceEndpointRegistry_FindClient: rv}, nil
}

func (t *debugNetworkServiceEndpointRegistryClient) Unregister(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*empty.Empty, error) {
	operation := typeutils.GetFuncName(t.debugged, methodNameUnregister)
	updatedContext := withLog(ctx)

	if updatedContext.Value(nseClientUnregisterLoggedKey) == nil {
		updatedContext = context.WithValue(updatedContext, nseClientUnregisterLoggedKey, t)
		logObjectDebug(updatedContext, "nse-unregister", in)
	}

	// Actually call the next
	rv, err := t.debugged.Unregister(updatedContext, in, opts...)
	if err != nil {
		if updatedContext.Value(nseClientUnregisterLoggedKey) == t {
			return nil, logError(updatedContext, err, operation)
		}
		return nil, err
	}

	if updatedContext.Value(nseClientUnregisterLoggedKey) == t {
		logObjectDebug(updatedContext, "nse-unregister-response", rv)
	}
	return rv, nil
}

// NewNetworkServiceEndpointRegistryClient - wraps registry.NetworkServiceEndpointRegistryClient with tracing
func NewNetworkServiceEndpointRegistryClient(debugged registry.NetworkServiceEndpointRegistryClient) registry.NetworkServiceEndpointRegistryClient {
	return &debugNetworkServiceEndpointRegistryClient{debugged: debugged}
}

type debugNetworkServiceEndpointRegistryServer struct {
	debugged registry.NetworkServiceEndpointRegistryServer
}

func (t *debugNetworkServiceEndpointRegistryServer) Register(ctx context.Context, in *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	operation := typeutils.GetFuncName(t.debugged, methodNameRegister)
	updatedContext := withLog(ctx)

	if updatedContext.Value(nseServerRegisterLoggedKey) == nil {
		updatedContext = context.WithValue(updatedContext, nseServerRegisterLoggedKey, t)
		logObjectDebug(updatedContext, "nse-server-register", in)
	}

	rv, err := t.debugged.Register(updatedContext, in)
	if err != nil {
		if updatedContext.Value(nseServerRegisterLoggedKey) == t {
			return nil, logError(updatedContext, err, operation)
		}
		return nil, err
	}

	if updatedContext.Value(nseServerRegisterLoggedKey) == t {
		logObjectDebug(updatedContext, "nse-server-register-response", rv)
	}
	return rv, nil
}

func (t *debugNetworkServiceEndpointRegistryServer) Find(in *registry.NetworkServiceEndpointQuery, s registry.NetworkServiceEndpointRegistry_FindServer) error {
	operation := typeutils.GetFuncName(t.debugged, methodNameFind)
	updatedContext := withLog(s.Context())

	if updatedContext.Value(nseServerFindLoggedKey) == nil {
		updatedContext = context.WithValue(updatedContext, nseServerFindLoggedKey, t)
		logObjectDebug(updatedContext, "nse-server-find", in)
	}

	s = &debugNetworkServiceEndpointRegistryFindServer{
		NetworkServiceEndpointRegistry_FindServer: streamcontext.NetworkServiceEndpointRegistryFindServer(updatedContext, s),
	}

	// Actually call the next
	err := t.debugged.Find(in, s)
	if err != nil {
		if updatedContext.Value(nseServerRegisterLoggedKey) == t {
			return logError(updatedContext, err, operation)
		}
		return err
	}

	if updatedContext.Value(nseServerRegisterLoggedKey) == t {
		logObjectDebug(updatedContext, "nse-server-find-response", in)
	}

	return nil
}

func (t *debugNetworkServiceEndpointRegistryServer) Unregister(ctx context.Context, in *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	operation := typeutils.GetFuncName(t.debugged, methodNameUnregister)
	updatedContext := withLog(ctx)

	if updatedContext.Value(nseServerUnregisterLoggedKey) == nil {
		updatedContext = context.WithValue(updatedContext, nseServerUnregisterLoggedKey, t)
		logObjectDebug(updatedContext, "nse-server-unregister", in)
	}

	// Actually call the next
	rv, err := t.debugged.Unregister(updatedContext, in)
	if err != nil {
		if updatedContext.Value(nseServerUnregisterLoggedKey) == t {
			return nil, logError(updatedContext, err, operation)
		}
		return nil, err
	}

	if updatedContext.Value(nseServerUnregisterLoggedKey) == t {
		logObjectDebug(updatedContext, "nse-server-unregister-response", rv)
	}
	return rv, nil
}

// NewNetworkServiceEndpointRegistryServer - wraps registry.NetworkServiceEndpointRegistryServer with tracing
func NewNetworkServiceEndpointRegistryServer(debugged registry.NetworkServiceEndpointRegistryServer) registry.NetworkServiceEndpointRegistryServer {
	return &debugNetworkServiceEndpointRegistryServer{debugged: debugged}
}

type debugNetworkServiceEndpointRegistryFindServer struct {
	registry.NetworkServiceEndpointRegistry_FindServer
}

func (t *debugNetworkServiceEndpointRegistryFindServer) Send(nseResp *registry.NetworkServiceEndpointResponse) error {
	operation := typeutils.GetFuncName(t.NetworkServiceEndpointRegistry_FindServer, methodNameSend)
	updatedContext := withLog(t.Context())

	if isReadyForLogging(updatedContext, false) && storeNSEServerSendLogged(updatedContext) {
		logObjectDebug(updatedContext, "network service endpoint", nseResp.NetworkServiceEndpoint)
	}

	s := streamcontext.NetworkServiceEndpointRegistryFindServer(updatedContext, t.NetworkServiceEndpointRegistry_FindServer)
	err := s.Send(nseResp)
	if err != nil {
		if isReadyForLogging(updatedContext, false) && storeNSEServerSendErrorLogged(updatedContext) {
			return logError(updatedContext, err, operation)
		}

		return err
	}
	return nil
}
