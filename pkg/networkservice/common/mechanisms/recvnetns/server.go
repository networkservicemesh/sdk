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

package recvnetns

import (
	"context"

	"github.com/edwarnicke/grpcfd"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"google.golang.org/grpc/peer"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type recvNetnsServer struct {
	netNSFileMap FileMap
}

// NewServer - receives the netNS file from the Client (if kernel) mechanism and stores it in the context for retrieval
// with Filename(ctx)
func NewServer() networkservice.NetworkServiceServer {
	return &recvNetnsServer{}
}

func (n *recvNetnsServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	// If not a kernel mechanism... we don't have anything to do go on to the next server
	m := kernel.ToMechanism(request.GetConnection().GetMechanism())
	if m == nil {
		return next.Server(ctx).Request(ctx, request)
	}

	// If we already have a netnsFile ... just use that
	netnsFile, found := n.netNSFileMap.Load(request.GetConnection().GetId())
	if found {
		WithFilename(ctx, netnsFile.Name())
		return next.Server(ctx).Request(ctx, request)
	}

	p, ok := peer.FromContext(ctx)
	if !ok {
		return nil, errors.New("unable to extract a peer.Peer from context. Use grpc.Cred(grpcfd.TransportCredentials(...)) as grpc.ServerOption")
	}

	// If we don't have a recv
	recv, ok := p.Addr.(grpcfd.FDRecver)
	if !ok {
		return nil, errors.Errorf("unable to extract a grpc.FDSender from peer.Addr.  Use grpc.Cred(grpcfd.TransportCredentials(...)) as grpc.ServerOption. p.Addr: %+v", p.Addr)
	}
	// Get the file
	fileCh, err := recv.RecvFileByURL(m.GetNetNSURL())
	if err != nil {
		return nil, err
	}
	select {
	case <-ctx.Done():
		return nil, errors.WithStack(ctx.Err())
	case createdNetNSFile := <-fileCh:
		var loaded bool
		netnsFile, loaded = n.netNSFileMap.LoadOrStore(request.GetConnection().GetId(), createdNetNSFile)
		if loaded {
			_ = createdNetNSFile.Close()
		}
		ctx = WithFilename(ctx, netnsFile.Name())
		return next.Server(ctx).Request(ctx, request)
	}
}

func (n *recvNetnsServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	netnsFile, found := n.netNSFileMap.Load(conn.GetId())
	if found {
		ctx = WithFilename(ctx, netnsFile.Name())
	}
	return next.Server(ctx).Close(ctx, conn)
}
