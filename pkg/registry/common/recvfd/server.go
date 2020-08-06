// Copyright (c) 2020 Cisco and/or its affiliates.
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

// +build !windows

// Package recvfd provides an NSE registry server chain element that:
//  1. Receives and fd over a unix file socket if the nse.URL is an inode://${dev}/${inode} url
//  2. Rewrites the nse.URL to unix:///proc/${pid}/fd/${fd} so it can be used by a normal dialer
package recvfd

import (
	"context"
	"fmt"
	"os"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc/peer"

	"github.com/edwarnicke/grpcfd"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/serialize"
)

type recvfdNseServer struct {
	filesByNSEName map[string]*os.File
	executor       serialize.Executor
}

// NewNetworkServiceEndpointRegistryServer - creates new NSE registry chain element that will:
//  1. Receive and fd over a unix file socket if the nse.URL is an inode://${dev}/${inode} url
//  2. Rewrite the nse.URL to unix:///proc/${pid}/fd/${fd} so it can be used by a normal dialer
func NewNetworkServiceEndpointRegistryServer() registry.NetworkServiceEndpointRegistryServer {
	return &recvfdNseServer{
		filesByNSEName: make(map[string]*os.File),
	}
}

func (r *recvfdNseServer) Register(ctx context.Context, endpoint *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	p, ok := peer.FromContext(ctx)
	_, _, err := grpcfd.URLStringToDevIno(endpoint.Url)
	if ok && p.Addr != nil && err == nil {
		if recv, ok := p.Addr.(grpcfd.FDRecver); ok {
			fileCh, err := recv.RecvFileByURL(endpoint.Url)
			if err != nil {
				return nil, err
			}
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case file := <-fileCh:
				r.executor.AsyncExec(func() {
					r.filesByNSEName[endpoint.Name] = file
				})
				endpoint = endpoint.Clone()
				endpoint.Url = fmt.Sprintf("unix://%s", file.Name())
			}
		}
	}
	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, endpoint)
}

func (r *recvfdNseServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, server)
}

func (r *recvfdNseServer) Unregister(ctx context.Context, endpoint *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	r.executor.AsyncExec(func() {
		// Note: intentionally not closing the file here as it will be closed by its finalizer when its no longer referenced
		//       and GC runs.  Since we *only* kept the file reference here to prevent the file being GCed and Closed... we don't
		//       need to protect it anymore
		delete(r.filesByNSEName, endpoint.Name)
	})
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, endpoint)
}
