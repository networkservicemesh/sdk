// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
//
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

// Package streamchannel provides find client/servers based on channels
package streamchannel

import (
	"context"
	"io"

	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

// NewNetworkServiceEndpointFindClient creates NetworkServiceEndpointRegistry_FindClient based on passed channel.
func NewNetworkServiceEndpointFindClient(ctx context.Context, recvCh <-chan *registry.NetworkServiceEndpointResponse) registry.NetworkServiceEndpointRegistry_FindClient {
	return &networkServiceEndpointRegistryFindClient{
		ctx:    ctx,
		recvCh: recvCh,
	}
}

type networkServiceEndpointRegistryFindClient struct {
	grpc.ClientStream
	err    error
	recvCh <-chan *registry.NetworkServiceEndpointResponse
	ctx    context.Context
}

func (c *networkServiceEndpointRegistryFindClient) Recv() (*registry.NetworkServiceEndpointResponse, error) {
	res, ok := <-c.recvCh

	if !ok {
		err := io.EOF
		if c.err == nil {
			return nil, err
		}
		return res, errors.Wrap(c.err, err.Error())
	}
	return res, errors.WithStack(c.err)
}

func (c *networkServiceEndpointRegistryFindClient) Context() context.Context {
	return c.ctx
}

var _ registry.NetworkServiceEndpointRegistry_FindClient = &networkServiceEndpointRegistryFindClient{}

// NewNetworkServiceEndpointFindServer creates NetworkServiceEndpointRegistry_FindServer based on passed channel.
func NewNetworkServiceEndpointFindServer(ctx context.Context, sendCh chan<- *registry.NetworkServiceEndpointResponse) registry.NetworkServiceEndpointRegistry_FindServer {
	return &networkServiceEndpointRegistryFindServer{
		ctx:    ctx,
		sendCh: sendCh,
	}
}

type networkServiceEndpointRegistryFindServer struct {
	grpc.ServerStream
	ctx    context.Context
	sendCh chan<- *registry.NetworkServiceEndpointResponse
}

func (s *networkServiceEndpointRegistryFindServer) Send(nseResp *registry.NetworkServiceEndpointResponse) error {
	select {
	case <-s.ctx.Done():
		return errors.Wrap(s.ctx.Err(), "application context is done")
	case s.sendCh <- nseResp:
		return nil
	}
}

func (s *networkServiceEndpointRegistryFindServer) Context() context.Context {
	return s.ctx
}

var _ registry.NetworkServiceEndpointRegistry_FindServer = &networkServiceEndpointRegistryFindServer{}
