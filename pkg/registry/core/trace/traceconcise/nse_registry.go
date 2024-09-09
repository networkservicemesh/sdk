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

package traceconcise

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

type conciseNetworkServiceEndpointRegistryClient struct {
	traced registry.NetworkServiceEndpointRegistryClient
}

type conciseNetworkServiceEndpointRegistryFindClient struct {
	registry.NetworkServiceEndpointRegistry_FindClient
	operation string
}

func (t *conciseNetworkServiceEndpointRegistryFindClient) Recv() (*registry.NetworkServiceEndpointResponse, error) {
	ctx := t.Context()

	s := streamcontext.NetworkServiceEndpointRegistryFindClient(ctx, t.NetworkServiceEndpointRegistry_FindClient)
	rv, err := s.Recv()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, errors.WithStack(err)
		}
		if status.Code(err) == codes.Canceled {
			return nil, errors.WithStack(err)
		}

		lastError := loadAndStoreNseClientRecvError(ctx, err)
		if lastError == nil || err.Error() != lastError.Error() {
			return nil, logError(ctx, err, t.operation)
		}
		return nil, err
	}

	ctx, tail := nseClientFindTail(ctx)
	if tail == t {
		logObject(ctx, "nse-client-recv-response", rv)
	}

	ctx, head := nseClientFindHead(ctx)
	if head == t {
		logObject(ctx, "nse-client-recv", rv)
	}
	return rv, nil
}

func (t *conciseNetworkServiceEndpointRegistryClient) Register(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	ctx, tail := nseClientRegisterTail(ctx)
	if tail == nil {
		var finish func()
		ctx, finish = withLog(ctx, methodNameRegister)
		defer finish()
		logObject(ctx, "nse-client-register", in)
	}
	ctx = withNseClientRegisterTail(ctx, t.traced)

	rv, err := t.traced.Register(ctx, in, opts...)
	if err != nil {
		lastError := loadAndStoreNseClientRegisterError(ctx, err)
		if lastError == nil || err.Error() != lastError.Error() {
			operation := typeutils.GetFuncName(t.traced, methodNameRegister)
			return nil, logError(ctx, err, operation)
		}
		return nil, err
	}

	ctx, tail = nseClientRegisterTail(ctx)
	if tail == t.traced {
		logObject(ctx, "nse-client-register-response", rv)
	}
	return rv, nil
}

func (t *conciseNetworkServiceEndpointRegistryClient) Find(ctx context.Context, in *registry.NetworkServiceEndpointQuery, opts ...grpc.CallOption) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	conciseFindClient := &conciseNetworkServiceEndpointRegistryFindClient{
		operation: typeutils.GetFuncName(t.traced, methodNameRecv),
	}

	ctx, tail := nseClientFindTail(ctx)
	if tail == nil {
		var finish func()
		ctx, finish = withLog(ctx, methodNameFind)
		defer finish()
		logObject(ctx, "nse-client-find", in)
		ctx = withNseClientFindHead(ctx, conciseFindClient)
	}
	ctx = withNseClientFindTail(ctx, conciseFindClient)

	// Actually call the next
	rv, err := t.traced.Find(ctx, in, opts...)
	if err != nil {
		lastError := loadAndStoreNseClientFindError(ctx, err)
		if lastError == nil || err.Error() != lastError.Error() {
			operation := typeutils.GetFuncName(t.traced, methodNameFind)
			return nil, logError(ctx, err, operation)
		}
		return nil, err
	}

	ctx, tail = nseClientFindTail(ctx)
	if tail == conciseFindClient {
		logObject(ctx, "nse-client-find-response", rv)
	}

	conciseFindClient.NetworkServiceEndpointRegistry_FindClient = rv
	return conciseFindClient, nil
}

func (t *conciseNetworkServiceEndpointRegistryClient) Unregister(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*empty.Empty, error) {
	ctx, tail := nseClientUnregisterTail(ctx)
	if tail == nil {
		var finish func()
		ctx, finish = withLog(ctx, methodNameUnregister)
		defer finish()
		logObject(ctx, "nse-client-unregister", in)
	}
	ctx = withNseClientUnregisterTail(ctx, t.traced)

	// Actually call the next
	rv, err := t.traced.Unregister(ctx, in, opts...)
	if err != nil {
		lastError := loadAndStoreNseClientUnregisterError(ctx, err)
		if lastError == nil || err.Error() != lastError.Error() {
			operation := typeutils.GetFuncName(t.traced, methodNameUnregister)
			return nil, logError(ctx, err, operation)
		}
		return nil, err
	}
	ctx, tail = nseClientUnregisterTail(ctx)
	if tail == t.traced {
		logObject(ctx, "nse-client-unregister-response", in)
	}

	return rv, nil
}

// NewNetworkServiceEndpointRegistryClient - wraps registry.NetworkServiceEndpointRegistryClient with tracing.
func NewNetworkServiceEndpointRegistryClient(traced registry.NetworkServiceEndpointRegistryClient) registry.NetworkServiceEndpointRegistryClient {
	return &conciseNetworkServiceEndpointRegistryClient{traced: traced}
}

type conciseNetworkServiceEndpointRegistryServer struct {
	traced registry.NetworkServiceEndpointRegistryServer
}

func (t *conciseNetworkServiceEndpointRegistryServer) Register(ctx context.Context, in *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	ctx, tail := nseServerRegisterTail(ctx)
	if tail == nil {
		var finish func()
		ctx, finish = withLog(ctx, methodNameRegister)
		defer finish()
		logObject(ctx, "nse-server-register", in)
	}
	ctx = withNseServerRegisterTail(ctx, t.traced)

	rv, err := t.traced.Register(ctx, in)
	if err != nil {
		lastError := loadAndStoreNseServerRegisterError(ctx, err)
		if lastError == nil || err.Error() != lastError.Error() {
			operation := typeutils.GetFuncName(t.traced, methodNameRegister)
			return nil, logError(ctx, err, operation)
		}
		return nil, err
	}

	ctx, tail = nseServerRegisterTail(ctx)
	if tail == t.traced {
		logObject(ctx, "nse-server-register-response", rv)
	}
	return rv, nil
}

func (t *conciseNetworkServiceEndpointRegistryServer) Find(in *registry.NetworkServiceEndpointQuery, s registry.NetworkServiceEndpointRegistry_FindServer) error {
	conciseFindServer := &conciseNetworkServiceEndpointRegistryFindServer{
		operation: typeutils.GetFuncName(t.traced, methodNameSend),
	}

	ctx, tail := nseServerFindTail(s.Context())
	if tail == nil {
		var finish func()
		ctx, finish = withLog(s.Context(), methodNameFind)
		defer finish()
		logObject(ctx, "nse-server-find", in)
		ctx = withNseServerFindHead(ctx, conciseFindServer)
	}
	ctx = withNseServerFindTail(ctx, conciseFindServer)

	conciseFindServer.NetworkServiceEndpointRegistry_FindServer = streamcontext.NetworkServiceEndpointRegistryFindServer(ctx, s)

	// Actually call the next
	err := t.traced.Find(in, conciseFindServer)
	if err != nil {
		lastError := loadAndStoreNseServerFindError(ctx, err)
		if lastError == nil || err.Error() != lastError.Error() {
			operation := typeutils.GetFuncName(t.traced, methodNameFind)
			return logError(ctx, err, operation)
		}
		return err
	}

	ctx, tail = nseServerFindTail(ctx)
	if tail == conciseFindServer {
		logObject(ctx, "nse-server-find-response", in)
	}
	return nil
}

func (t *conciseNetworkServiceEndpointRegistryServer) Unregister(ctx context.Context, in *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	ctx, tail := nseServerUnregisterTail(ctx)
	if tail == nil {
		var finish func()
		ctx, finish = withLog(ctx, methodNameUnregister)
		defer finish()
		logObject(ctx, "nse-server-unregister", in)
	}
	ctx = withNseServerUnregisterTail(ctx, t.traced)

	// Actually call the next
	rv, err := t.traced.Unregister(ctx, in)
	if err != nil {
		lastError := loadAndStoreNseServerUnregisterError(ctx, err)
		if lastError == nil || err.Error() != lastError.Error() {
			operation := typeutils.GetFuncName(t.traced, methodNameUnregister)
			return nil, logError(ctx, err, operation)
		}
		return nil, err
	}

	ctx, tail = nseServerUnregisterTail(ctx)
	if tail == t.traced {
		logObject(ctx, "nse-server-unregister-response", in)
	}
	return rv, nil
}

// NewNetworkServiceEndpointRegistryServer - wraps registry.NetworkServiceEndpointRegistryServer with tracing.
func NewNetworkServiceEndpointRegistryServer(traced registry.NetworkServiceEndpointRegistryServer) registry.NetworkServiceEndpointRegistryServer {
	return &conciseNetworkServiceEndpointRegistryServer{traced: traced}
}

type conciseNetworkServiceEndpointRegistryFindServer struct {
	registry.NetworkServiceEndpointRegistry_FindServer
	operation string
}

func (t *conciseNetworkServiceEndpointRegistryFindServer) Send(nseResp *registry.NetworkServiceEndpointResponse) error {
	ctx, tail := nseServerFindTail(t.Context())
	if tail == t {
		logObject(ctx, "nse-server-send", nseResp)
	}

	s := streamcontext.NetworkServiceEndpointRegistryFindServer(ctx, t.NetworkServiceEndpointRegistry_FindServer)
	err := s.Send(nseResp)
	if err != nil {
		lastError := loadAndStoreNseServerSendError(ctx, err)
		if lastError == nil || err.Error() != lastError.Error() {
			return logError(ctx, err, t.operation)
		}
	}

	ctx, head := nseServerFindHead(ctx)
	if head == t {
		logObject(ctx, "nse-server-send-response", nseResp)
	}
	return err
}
