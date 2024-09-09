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

package streamchannel

import (
	"context"
	"io"

	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

// NewNetworkServiceFindClient creates new NetworkServiceRegistry_FindClient.
func NewNetworkServiceFindClient(ctx context.Context, recvCh <-chan *registry.NetworkServiceResponse) registry.NetworkServiceRegistry_FindClient {
	return &networkServiceRegistryFindClient{
		ctx:    ctx,
		recvCh: recvCh,
	}
}

type networkServiceRegistryFindClient struct {
	grpc.ClientStream
	err    error
	recvCh <-chan *registry.NetworkServiceResponse
	ctx    context.Context
}

func (c *networkServiceRegistryFindClient) Recv() (*registry.NetworkServiceResponse, error) {
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

func (c *networkServiceRegistryFindClient) Context() context.Context {
	return c.ctx
}

var _ registry.NetworkServiceRegistry_FindClient = &networkServiceRegistryFindClient{}

// NewNetworkServiceFindServer creates new NetworkServiceRegistry_FindServer based on passed channel.
func NewNetworkServiceFindServer(ctx context.Context, sendCh chan<- *registry.NetworkServiceResponse) registry.NetworkServiceRegistry_FindServer {
	return &networkServiceRegistryFindServer{
		ctx:    ctx,
		sendCh: sendCh,
	}
}

type networkServiceRegistryFindServer struct {
	grpc.ServerStream
	ctx    context.Context
	sendCh chan<- *registry.NetworkServiceResponse
}

func (s *networkServiceRegistryFindServer) Send(nsResp *registry.NetworkServiceResponse) error {
	select {
	case <-s.ctx.Done():
		return errors.Wrap(s.ctx.Err(), "application context is done")
	case s.sendCh <- nsResp:
		return nil
	}
}

func (s *networkServiceRegistryFindServer) Context() context.Context {
	return s.ctx
}

var _ registry.NetworkServiceRegistry_FindServer = &networkServiceRegistryFindServer{}
