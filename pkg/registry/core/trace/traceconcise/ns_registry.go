// Copyright (c) 2023 Cisco and/or its affiliates.
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

// Package traceconcise provides a wrapper for concise tracing around a registry.{Registry,Discovery}{Server,Client}
package traceconcise

import (
	"context"
	"io"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/streamcontext"
	"github.com/networkservicemesh/sdk/pkg/tools/typeutils"
)

type conciseNetworkServiceRegistryClient struct {
	traced registry.NetworkServiceRegistryClient
}

type conciseNetworkServiceRegistryFindClient struct {
	registry.NetworkServiceRegistry_FindClient
	operation string
}

func (t *conciseNetworkServiceRegistryFindClient) Recv() (*registry.NetworkServiceResponse, error) {
	ctx := t.Context()

	s := streamcontext.NetworkServiceRegistryFindClient(ctx, t.NetworkServiceRegistry_FindClient)
	rv, err := s.Recv()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, errors.WithStack(err)
		}
		if status.Code(err) == codes.Canceled {
			return nil, errors.WithStack(err)
		}

		lastError := loadAndStoreNsClientRecvError(ctx, err)
		if lastError == nil || err.Error() != lastError.Error() {
			return nil, logError(ctx, err, t.operation)
		}
		return nil, err
	}

	ctx, head := nsClientFindHead(ctx)
	if head == t {
		logObject(ctx, "ns-client-recv", rv)
	}

	ctx, tail := nsClientFindTail(ctx)
	if tail == t {
		logObject(ctx, "ns-client-recv-response", rv)
	}
	return rv, nil
}

func (t *conciseNetworkServiceRegistryClient) Register(ctx context.Context, in *registry.NetworkService, opts ...grpc.CallOption) (*registry.NetworkService, error) {
	ctx, tail := nsClientRegisterTail(ctx)
	if tail == nil {
		var finish func()
		ctx, finish = withLog(ctx, methodNameRegister)
		defer finish()
		logObject(ctx, "ns-client-register", in)
	}
	ctx = withNsClientRegisterTail(ctx, t.traced)

	rv, err := t.traced.Register(ctx, in, opts...)
	if err != nil {
		lastError := loadAndStoreNsClientRegisterError(ctx, err)
		if lastError == nil || err.Error() != lastError.Error() {
			operation := typeutils.GetFuncName(t.traced, methodNameRegister)
			return nil, logError(ctx, err, operation)
		}
		return nil, err
	}

	ctx, tail = nsClientRegisterTail(ctx)
	if tail == t.traced {
		logObject(ctx, "ns-client-register-response", rv)
	}
	return rv, nil
}

func (t *conciseNetworkServiceRegistryClient) Find(ctx context.Context, in *registry.NetworkServiceQuery, opts ...grpc.CallOption) (registry.NetworkServiceRegistry_FindClient, error) {
	conciseFindClient := &conciseNetworkServiceRegistryFindClient{
		operation: typeutils.GetFuncName(t.traced, methodNameRecv),
	}

	ctx, tail := nsClientFindTail(ctx)
	if tail == nil {
		var finish func()
		ctx, finish = withLog(ctx, methodNameFind)
		defer finish()
		logObject(ctx, "ns-client-find", in)
		ctx = withNsClientFindHead(ctx, conciseFindClient)
	}
	ctx = withNsClientFindTail(ctx, conciseFindClient)

	// Actually call the next
	rv, err := t.traced.Find(ctx, in, opts...)
	if err != nil {
		lastError := loadAndStoreNsClientFindError(ctx, err)
		if lastError == nil || err.Error() != lastError.Error() {
			operation := typeutils.GetFuncName(t.traced, methodNameFind)
			return nil, logError(ctx, err, operation)
		}
		return nil, err
	}

	ctx, tail = nsClientFindTail(ctx)
	if tail == conciseFindClient {
		logObject(ctx, "ns-client-find-response", rv)
	}

	conciseFindClient.NetworkServiceRegistry_FindClient = rv
	return conciseFindClient, nil
}

func (t *conciseNetworkServiceRegistryClient) Unregister(ctx context.Context, in *registry.NetworkService, opts ...grpc.CallOption) (*empty.Empty, error) {
	ctx, tail := nsClientUnregisterTail(ctx)
	if tail == nil {
		var finish func()
		ctx, finish = withLog(ctx, methodNameUnregister)
		defer finish()
		logObject(ctx, "ns-client-unregister", in)
	}
	ctx = withNsClientUnregisterTail(ctx, t.traced)

	// Actually call the next
	rv, err := t.traced.Unregister(ctx, in, opts...)
	if err != nil {
		lastError := loadAndStoreNsClientUnregisterError(ctx, err)
		if lastError == nil || err.Error() != lastError.Error() {
			operation := typeutils.GetFuncName(t.traced, methodNameUnregister)
			return nil, logError(ctx, err, operation)
		}
		return nil, err
	}
	ctx, tail = nsClientUnregisterTail(ctx)
	if tail == t.traced {
		logObject(ctx, "ns-client-unregister-response", in)
	}

	return rv, nil
}

// NewNetworkServiceRegistryClient - wraps registry.NetworkServiceRegistryClient with tracing.
func NewNetworkServiceRegistryClient(traced registry.NetworkServiceRegistryClient) registry.NetworkServiceRegistryClient {
	return &conciseNetworkServiceRegistryClient{traced: traced}
}

type conciseNetworkServiceRegistryServer struct {
	traced registry.NetworkServiceRegistryServer
}

func (t *conciseNetworkServiceRegistryServer) Register(ctx context.Context, in *registry.NetworkService) (*registry.NetworkService, error) {
	ctx, tail := nsServerRegisterTail(ctx)
	if tail == nil {
		var finish func()
		ctx, finish = withLog(ctx, methodNameRegister)
		defer finish()
		logObject(ctx, "ns-server-register", in)
	}
	ctx = withNsServerRegisterTail(ctx, t.traced)

	rv, err := t.traced.Register(ctx, in)
	if err != nil {
		lastError := loadAndStoreNsServerRegisterError(ctx, err)
		if lastError == nil || err.Error() != lastError.Error() {
			operation := typeutils.GetFuncName(t.traced, methodNameRegister)
			return nil, logError(ctx, err, operation)
		}
		return nil, err
	}

	ctx, tail = nsServerRegisterTail(ctx)
	if tail == t.traced {
		logObject(ctx, "ns-server-register-response", rv)
	}
	return rv, nil
}

func (t *conciseNetworkServiceRegistryServer) Find(in *registry.NetworkServiceQuery, s registry.NetworkServiceRegistry_FindServer) error {
	conciseFindServer := &conciseNetworkServiceRegistryFindServer{
		operation: typeutils.GetFuncName(t.traced, methodNameSend),
	}

	ctx, tail := nsServerFindTail(s.Context())
	if tail == nil {
		var finish func()
		ctx, finish = withLog(ctx, methodNameFind)
		defer finish()
		logObject(ctx, "ns-server-find", in)
		ctx = withNsServerFindHead(ctx, conciseFindServer)
	}
	ctx = withNsServerFindTail(ctx, conciseFindServer)

	conciseFindServer.NetworkServiceRegistry_FindServer = streamcontext.NetworkServiceRegistryFindServer(ctx, s)

	// Actually call the next
	err := t.traced.Find(in, conciseFindServer)
	if err != nil {
		lastError := loadAndStoreNsServerFindError(ctx, err)
		if lastError == nil || err.Error() != lastError.Error() {
			operation := typeutils.GetFuncName(t.traced, methodNameFind)
			return logError(ctx, err, operation)
		}
		return err
	}

	ctx, tail = nsServerFindTail(ctx)
	if tail == conciseFindServer {
		logObject(ctx, "ns-server-find-response", in)
	}
	return nil
}

func (t *conciseNetworkServiceRegistryServer) Unregister(ctx context.Context, in *registry.NetworkService) (*empty.Empty, error) {
	ctx, tail := nsServerUnregisterTail(ctx)
	if tail == nil {
		var finish func()
		ctx, finish = withLog(ctx, methodNameUnregister)
		defer finish()
		logObject(ctx, "ns-server-unregister", in)
	}
	ctx = withNsServerUnregisterTail(ctx, t.traced)

	// Actually call the next
	rv, err := t.traced.Unregister(ctx, in)
	if err != nil {
		lastError := loadAndStoreNsServerUnregisterError(ctx, err)
		if lastError == nil || err.Error() != lastError.Error() {
			operation := typeutils.GetFuncName(t.traced, methodNameUnregister)
			return nil, logError(ctx, err, operation)
		}
		return nil, err
	}

	ctx, tail = nsServerUnregisterTail(ctx)
	if tail == t.traced {
		logObject(ctx, "ns-server-unregister-response", in)
	}
	return rv, nil
}

// NewNetworkServiceRegistryServer - wraps registry.NetworkServiceRegistryServer with tracing.
func NewNetworkServiceRegistryServer(traced registry.NetworkServiceRegistryServer) registry.NetworkServiceRegistryServer {
	return &conciseNetworkServiceRegistryServer{traced: traced}
}

type conciseNetworkServiceRegistryFindServer struct {
	registry.NetworkServiceRegistry_FindServer
	operation string
}

func (t *conciseNetworkServiceRegistryFindServer) Send(nsResp *registry.NetworkServiceResponse) error {
	ctx, tail := nsServerFindTail(t.Context())
	if tail == t {
		logObject(ctx, "ns-server-send", nsResp)
	}

	s := streamcontext.NetworkServiceRegistryFindServer(ctx, t.NetworkServiceRegistry_FindServer)
	err := s.Send(nsResp)
	if err != nil {
		lastError := loadAndStoreNsServerSendError(ctx, err)
		if lastError == nil || err.Error() != lastError.Error() {
			return logError(ctx, err, t.operation)
		}
	}

	ctx, head := nsServerFindHead(ctx)
	if head == t {
		logObject(ctx, "ns-server-send-response", nsResp)
	}
	return err
}
