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
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type recvNetnsClient struct {
	netNSFileMap FileMap
}

// NewClient - return new client chain element that recvs the netNS (if kernel mechanism) from the downstream NSE and stores its filename
// in the *string provided to WithFilenamePtr
func NewClient() networkservice.NetworkServiceClient {
	return &recvNetnsClient{}
}

func (n *recvNetnsClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	// Call the next client asking for a peer
	p := peer.Peer{}
	conn, err := next.Client(ctx).Request(ctx, request, append(opts, grpc.Peer(&p))...)
	if err != nil {
		return nil, err
	}

	// If no client downstream of us is asking for a netns.Filename, we have nothing to do, so just return
	netnsFilePtr := FilenamePtr(ctx)
	if netnsFilePtr == nil {
		return conn, nil
	}

	// If not a kernel mechanism... we don't have anything to do, so return the result we got from the next client
	m := kernel.ToMechanism(conn.GetMechanism())
	if m == nil {
		return conn, nil
	}

	// If we already have a netnsFile ... just use that
	netnsFile, found := n.netNSFileMap.Load(conn.GetId())
	if found {
		*netnsFilePtr = netnsFile.Name()
		return conn, nil
	}

	// If we don't have a grpcfd.FDRecver, return an error
	recv, ok := p.Addr.(grpcfd.FDRecver)
	if !ok {
		return nil, errors.Errorf("unable to extract a grpc.FDRecv from peer.Addr.  Use grpc.Cred(grpcfd.TransportCredentials(...)) as grpc.ServerOption. peer.Addr: %+v", p.Addr)
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
		netnsFile, loaded = n.netNSFileMap.LoadOrStore(conn.GetId(), createdNetNSFile)
		if loaded {
			_ = createdNetNSFile.Close()
		}
		*netnsFilePtr = netnsFile.Name()
		return conn, nil
	}
}

func (n *recvNetnsClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	_, err := next.Client(ctx).Close(ctx, conn, opts...)
	if err != nil {
		return nil, err
	}
	netnsFilePtr := FilenamePtr(ctx)
	if netnsFilePtr != nil {
		netnsFile, found := n.netNSFileMap.Load(conn.GetId())
		if found {
			*netnsFilePtr = netnsFile.Name()
		}
	}
	return &empty.Empty{}, nil
}
