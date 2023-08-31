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

// Package debug provides a wrapper for debugging around a registry.{Registry,Discovery}{Server,Client}
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

type debugNetworkServiceRegistryClient struct {
	debugged registry.NetworkServiceRegistryClient
}

type debugNetworkServiceRegistryFindClient struct {
	registry.NetworkServiceRegistry_FindClient
}

const (
	nsClientRegisterLoggedKey   contextKeyType = "nsClientRegisterLoggedKey"
	nsClientFindLoggedKey       contextKeyType = "nsClientFindLoggedKey"
	nsClientUnregisterLoggedKey contextKeyType = "nsClientUnregisterLoggedKey"
	nsServerRegisterLoggedKey   contextKeyType = "nsServerRegisterLoggedKey"
	nsServerFindLoggedKey       contextKeyType = "nsServerFindLoggedKey"
	nsServerUnregisterLoggedKey contextKeyType = "nsServerUnregisterLoggedKey"
)

func (t *debugNetworkServiceRegistryFindClient) Recv() (*registry.NetworkServiceResponse, error) {
	operation := typeutils.GetFuncName(t.NetworkServiceRegistry_FindClient, methodNameRecv)
	updatedContext := withLog(t.Context())

	s := streamcontext.NetworkServiceRegistryFindClient(updatedContext, t.NetworkServiceRegistry_FindClient)
	rv, err := s.Recv()

	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, errors.WithStack(err)
		}
		if status.Code(err) == codes.Canceled {
			return nil, errors.WithStack(err)
		}

		if isReadyForLogging(updatedContext, true) && storeNSClientRecvErrorLogged(updatedContext) {
			return nil, logError(updatedContext, err, operation)
		}

		return nil, err
	}

	if isReadyForLogging(updatedContext, true) && storeNSClientRecvLogged(updatedContext) {
		logObjectDebug(updatedContext, "ns-recv-response", rv.NetworkService)
	}

	return rv, nil
}

func (t *debugNetworkServiceRegistryClient) Register(ctx context.Context, in *registry.NetworkService, opts ...grpc.CallOption) (*registry.NetworkService, error) {
	operation := typeutils.GetFuncName(t.debugged, methodNameRegister)
	updatedContext := withLog(ctx)

	if updatedContext.Value(nsClientRegisterLoggedKey) == nil {
		updatedContext = context.WithValue(updatedContext, nsClientRegisterLoggedKey, t)
		logObjectDebug(updatedContext, "ns-register", in)
	}

	rv, err := t.debugged.Register(updatedContext, in, opts...)
	if err != nil {
		if updatedContext.Value(nsClientRegisterLoggedKey) == t {
			return nil, logError(updatedContext, err, operation)
		}
		return nil, err
	}

	if updatedContext.Value(nsClientRegisterLoggedKey) == t {
		logObjectDebug(updatedContext, "ns-register-response", rv)
	}
	return rv, nil
}

func (t *debugNetworkServiceRegistryClient) Find(ctx context.Context, in *registry.NetworkServiceQuery, opts ...grpc.CallOption) (registry.NetworkServiceRegistry_FindClient, error) {
	operation := typeutils.GetFuncName(t.debugged, methodNameFind)
	updatedContext := withLog(ctx)

	if updatedContext.Value(nsClientFindLoggedKey) == nil {
		updatedContext = context.WithValue(updatedContext, nsClientFindLoggedKey, t)
		logObjectDebug(updatedContext, "ns-find", in)
	}

	// Actually call the next
	rv, err := t.debugged.Find(updatedContext, in, opts...)
	if err != nil {
		if updatedContext.Value(nsClientFindLoggedKey) == t {
			return nil, logError(updatedContext, err, operation)
		}
		return nil, err
	}

	if updatedContext.Value(nsClientFindLoggedKey) == t {
		logObjectDebug(updatedContext, "ns-find-response", rv)
	}
	return &debugNetworkServiceRegistryFindClient{NetworkServiceRegistry_FindClient: rv}, nil
}

func (t *debugNetworkServiceRegistryClient) Unregister(ctx context.Context, in *registry.NetworkService, opts ...grpc.CallOption) (*empty.Empty, error) {
	operation := typeutils.GetFuncName(t.debugged, methodNameUnregister)
	updatedContext := withLog(ctx)

	if updatedContext.Value(nsClientUnregisterLoggedKey) == nil {
		updatedContext = context.WithValue(updatedContext, nsClientUnregisterLoggedKey, t)
		logObjectDebug(updatedContext, "ns-unregister", in)
	}

	// Actually call the next
	rv, err := t.debugged.Unregister(updatedContext, in, opts...)
	if err != nil {
		if updatedContext.Value(nsClientUnregisterLoggedKey) == t {
			return nil, logError(updatedContext, err, operation)
		}
		return nil, err
	}

	if updatedContext.Value(nsClientUnregisterLoggedKey) == t {
		logObjectDebug(updatedContext, "ns-unregister-response", rv)
	}
	return rv, nil
}

// NewNetworkServiceRegistryClient - wraps registry.NetworkServiceRegistryClient with tracing
func NewNetworkServiceRegistryClient(debugged registry.NetworkServiceRegistryClient) registry.NetworkServiceRegistryClient {
	return &debugNetworkServiceRegistryClient{debugged: debugged}
}

type debugNetworkServiceRegistryServer struct {
	debugged registry.NetworkServiceRegistryServer
}

func (t *debugNetworkServiceRegistryServer) Register(ctx context.Context, in *registry.NetworkService) (*registry.NetworkService, error) {
	operation := typeutils.GetFuncName(t.debugged, methodNameRegister)
	updatedContext := withLog(ctx)

	if updatedContext.Value(nsServerRegisterLoggedKey) == nil {
		updatedContext = context.WithValue(updatedContext, nsServerRegisterLoggedKey, t)
		logObjectDebug(updatedContext, "ns-server-register", in)
	}

	rv, err := t.debugged.Register(updatedContext, in)
	if err != nil {
		if updatedContext.Value(nsServerRegisterLoggedKey) == t {
			return nil, logError(updatedContext, err, operation)
		}
		return nil, err
	}

	if updatedContext.Value(nsServerRegisterLoggedKey) == t {
		logObjectDebug(updatedContext, "ns-server-register-response", rv)
	}
	return rv, nil
}

func (t *debugNetworkServiceRegistryServer) Find(in *registry.NetworkServiceQuery, s registry.NetworkServiceRegistry_FindServer) error {
	operation := typeutils.GetFuncName(t.debugged, methodNameFind)
	updatedContext := withLog(s.Context())

	if updatedContext.Value(nsServerFindLoggedKey) == nil {
		updatedContext = context.WithValue(updatedContext, nsServerFindLoggedKey, t)
		logObjectDebug(updatedContext, "ns-server-find", in)
	}

	s = &debugNetworkServiceRegistryFindServer{
		NetworkServiceRegistry_FindServer: streamcontext.NetworkServiceRegistryFindServer(updatedContext, s),
	}

	// Actually call the next
	err := t.debugged.Find(in, s)
	if err != nil {
		if updatedContext.Value(nsServerFindLoggedKey) == t {
			return logError(updatedContext, err, operation)
		}

		return err
	}
	return nil
}

func (t *debugNetworkServiceRegistryServer) Unregister(ctx context.Context, in *registry.NetworkService) (*empty.Empty, error) {
	operation := typeutils.GetFuncName(t.debugged, methodNameUnregister)
	updatedContext := withLog(ctx)

	if updatedContext.Value(nsServerUnregisterLoggedKey) == nil {
		updatedContext = context.WithValue(updatedContext, nsServerUnregisterLoggedKey, t)
		logObjectDebug(updatedContext, "ns-server-unregister", in)
	}

	// Actually call the next
	rv, err := t.debugged.Unregister(updatedContext, in)
	if err != nil {
		if updatedContext.Value(nsServerUnregisterLoggedKey) == t {
			return nil, logError(updatedContext, err, operation)
		}
		return nil, err
	}

	if updatedContext.Value(nsServerUnregisterLoggedKey) == t {
		logObjectDebug(updatedContext, "ns-server-unregister-response", rv)
	}
	return rv, nil
}

// NewNetworkServiceRegistryServer - wraps registry.NetworkServiceRegistryServer with tracing
func NewNetworkServiceRegistryServer(debugged registry.NetworkServiceRegistryServer) registry.NetworkServiceRegistryServer {
	return &debugNetworkServiceRegistryServer{debugged: debugged}
}

type debugNetworkServiceRegistryFindServer struct {
	registry.NetworkServiceRegistry_FindServer
}

func (t *debugNetworkServiceRegistryFindServer) Send(nsResp *registry.NetworkServiceResponse) error {
	operation := typeutils.GetFuncName(t.NetworkServiceRegistry_FindServer, methodNameSend)
	updatedContext := withLog(t.Context())

	if isReadyForLogging(updatedContext, false) && storeNSServerSendLogged(updatedContext) {
		logObjectDebug(updatedContext, "network service", nsResp.NetworkService)
	}

	s := streamcontext.NetworkServiceRegistryFindServer(updatedContext, t.NetworkServiceRegistry_FindServer)
	err := s.Send(nsResp)
	if err != nil {
		if isReadyForLogging(updatedContext, false) && storeNSServerSendErrorLogged(updatedContext) {
			return logError(updatedContext, err, operation)
		}
		return err
	}

	return nil
}
