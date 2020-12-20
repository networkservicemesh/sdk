// Copyright (c) 2020 Doc.ai and/or its affiliates.
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

	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk/pkg/registry/core/streamcontext"
	"github.com/networkservicemesh/sdk/pkg/tools/logger"
	"github.com/networkservicemesh/sdk/pkg/tools/typeutils"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/registry"
)

type traceNetworkServiceEndpointRegistryClient struct {
	traced registry.NetworkServiceEndpointRegistryClient
}

type traceNetworkServiceEndpointRegistryFindClient struct {
	registry.NetworkServiceEndpointRegistry_FindClient
}

func (t *traceNetworkServiceEndpointRegistryFindClient) Recv() (*registry.NetworkServiceEndpoint, error) {
	operation := typeutils.GetFuncName(t.NetworkServiceEndpointRegistry_FindClient, "Recv")
	log, ctx, done := logger.NewLogger(t.Context(), operation)
	defer done(log)
	s := streamcontext.NetworkServiceEndpointRegistryFindClient(ctx, t.NetworkServiceEndpointRegistry_FindClient)
	rv, err := s.Recv()
	if err != nil {
		if _, ok := err.(stackTracer); !ok {
			err = errors.Wrapf(err, "Error returned from %s", operation)
			log.Errorf("%+v", err)
			return nil, err
		}
		log.Errorf("%v", err)
		return nil, err
	}
	log.Info("response", rv)
	return rv, err
}

func (t *traceNetworkServiceEndpointRegistryClient) Register(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	operation := typeutils.GetFuncName(t.traced, "Register")
	log, ctx, done := logger.NewLogger(ctx, operation)
	defer done(log)

	log.Info("request", in)

	rv, err := t.traced.Register(ctx, in, opts...)
	if err != nil {
		if _, ok := err.(stackTracer); !ok {
			err = errors.Wrapf(err, "Error returned from %s", operation)
			log.Errorf("%+v", err)
			return nil, err
		}
		log.Errorf("%v", err)
		return nil, err
	}
	log.Info("response", rv)
	return rv, err
}
func (t *traceNetworkServiceEndpointRegistryClient) Find(ctx context.Context, in *registry.NetworkServiceEndpointQuery, opts ...grpc.CallOption) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	operation := typeutils.GetFuncName(t.traced, "Find")
	log, ctx, done := logger.NewLogger(ctx, operation)
	defer done(log)

	log.Info("find", in)

	// Actually call the next
	rv, err := t.traced.Find(ctx, in, opts...)
	if err != nil {
		if _, ok := err.(stackTracer); !ok {
			err = errors.Wrapf(err, "Error returned from %s", operation)
			log.Errorf("%+v", err)
			return nil, err
		}
		log.Errorf("%v", err)
		return nil, err
	}
	log.Info("response", rv)

	return &traceNetworkServiceEndpointRegistryFindClient{NetworkServiceEndpointRegistry_FindClient: rv}, nil
}

func (t *traceNetworkServiceEndpointRegistryClient) Unregister(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*empty.Empty, error) {
	operation := typeutils.GetFuncName(t.traced, "Unregister")
	log, ctx, done := logger.NewLogger(ctx, operation)
	defer done(log)

	log.Info("request", in)

	// Actually call the next
	rv, err := t.traced.Unregister(ctx, in, opts...)
	if err != nil {
		if _, ok := err.(stackTracer); !ok {
			err = errors.Wrapf(err, "Error returned from %s", operation)
			log.Errorf("%+v", err)
			return nil, err
		}
		log.Errorf("%v", err)
		return nil, err
	}
	log.Info("response", rv)
	return rv, err
}

// NewNetworkServiceEndpointRegistryClient - wraps registry.NetworkServiceEndpointRegistryClient with tracing
func NewNetworkServiceEndpointRegistryClient(traced registry.NetworkServiceEndpointRegistryClient) registry.NetworkServiceEndpointRegistryClient {
	return &traceNetworkServiceEndpointRegistryClient{traced: traced}
}

type traceNetworkServiceEndpointRegistryServer struct {
	traced registry.NetworkServiceEndpointRegistryServer
}

func (t *traceNetworkServiceEndpointRegistryServer) Register(ctx context.Context, in *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	operation := typeutils.GetFuncName(t.traced, "Register")
	log, ctx, done := logger.NewLogger(ctx, operation)
	defer done(log)

	log.Info("request", in)

	rv, err := t.traced.Register(ctx, in)
	if err != nil {
		if _, ok := err.(stackTracer); !ok {
			err = errors.Wrapf(err, "Error returned from %s", operation)
			log.Errorf("%+v", err)
			return nil, err
		}
		log.Errorf("%v", err)
		return nil, err
	}
	log.Info("response", rv)
	return rv, err
}

func (t *traceNetworkServiceEndpointRegistryServer) Find(in *registry.NetworkServiceEndpointQuery, s registry.NetworkServiceEndpointRegistry_FindServer) error {
	operation := typeutils.GetFuncName(t.traced, "Find")
	log, ctx, done := logger.NewLogger(s.Context(), operation)
	defer done(log)
	s = &traceNetworkServiceEndpointRegistryFindServer{
		NetworkServiceEndpointRegistry_FindServer: streamcontext.NetworkServiceEndpointRegistryFindServer(ctx, s),
	}
	log.Info("find", in)

	// Actually call the next
	err := t.traced.Find(in, s)
	if err != nil {
		if _, ok := err.(stackTracer); !ok {
			err = errors.Wrapf(err, "Error returned from %s", operation)
			log.Errorf("%+v", err)
			return err
		}
		log.Errorf("%v", err)
		return err
	}
	return nil
}

func (t *traceNetworkServiceEndpointRegistryServer) Unregister(ctx context.Context, in *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	operation := typeutils.GetFuncName(t.traced, "Unregister")
	log, ctx, done := logger.NewLogger(ctx, operation)
	defer done(log)

	log.Info("request", in)

	// Actually call the next
	rv, err := t.traced.Unregister(ctx, in)
	if err != nil {
		if _, ok := err.(stackTracer); !ok {
			err = errors.Wrapf(err, "Error returned from %s", operation)
			log.Errorf("%+v", err)
			return nil, err
		}
		log.Errorf("%v", err)
		return nil, err
	}
	log.Info("response", rv)
	return rv, err
}

// NewNetworkServiceEndpointRegistryServer - wraps registry.NetworkServiceEndpointRegistryServer with tracing
func NewNetworkServiceEndpointRegistryServer(traced registry.NetworkServiceEndpointRegistryServer) registry.NetworkServiceEndpointRegistryServer {
	return &traceNetworkServiceEndpointRegistryServer{traced: traced}
}

type traceNetworkServiceEndpointRegistryFindServer struct {
	registry.NetworkServiceEndpointRegistry_FindServer
}

func (t *traceNetworkServiceEndpointRegistryFindServer) Send(nse *registry.NetworkServiceEndpoint) error {
	operation := typeutils.GetFuncName(t.NetworkServiceEndpointRegistry_FindServer, "Send")
	log, ctx, done := logger.NewLogger(t.Context(), operation)
	defer done(log)
	log.Info("network service endpoint", nse)
	s := streamcontext.NetworkServiceEndpointRegistryFindServer(ctx, t.NetworkServiceEndpointRegistry_FindServer)
	err := s.Send(nse)
	if err != nil {
		if _, ok := err.(stackTracer); !ok {
			err = errors.Wrapf(err, "Error returned from %s", operation)
			log.Errorf("%+v", err)
			return err
		}
		log.Errorf("%v", err)
		return err
	}
	return err
}
