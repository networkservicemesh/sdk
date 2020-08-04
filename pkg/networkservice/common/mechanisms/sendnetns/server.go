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

// Package sendnetns provides facilities to send the NetNS fd over local unix file sockets
package sendnetns

import (
	"context"
	"net/url"

	"github.com/edwarnicke/grpcfd"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"google.golang.org/grpc/peer"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type sendnsServer struct{}

// NewServer - receives the netNS file back to the Client (if kernel) mechanism and unix:///* Mechanism parameter is provided
// for kernel.NetNSURL
func NewServer() networkservice.NetworkServiceServer {
	return &sendnsServer{}
}

func (s *sendnsServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	// If not a kernel mechanism or we don't have a NetNSURL... we don't have anything to do go on to the next server
	m := kernel.ToMechanism(request.GetConnection().GetMechanism())
	if m == nil || m.GetNetNSURL() == "" {
		return next.Server(ctx).Request(ctx, request)
	}

	p, ok := peer.FromContext(ctx)
	if !ok {
		return nil, errors.New("unable to extract a peer.Peer from context. Make sure to use grpc.Cred(grpcfd.TransportCredentials(...)) as grpc.ServerOption")
	}
	sender, ok := p.Addr.(grpcfd.FDSender)
	if !ok {
		return nil, errors.Errorf("unable to extract a grpc.FDSender from peer.Addr.  Use grpc.Cred(grpcfd.TransportCredentials(...)) as grpc.ServerOption. p.Addr: %+v", p.Addr)
	}
	u, err := url.Parse(m.GetNetNSURL())
	if err != nil {
		return nil, err
	}
	if u.Scheme == "unix" {
		errCh := sender.SendFilename(u.Path)
		select {
		case err := <-errCh:
			if err != nil {
				return nil, err
			}
		default:
		}
		newURL, err := grpcfd.FilenameToURL(u.Path)
		if err != nil {
			return nil, err
		}
		request.GetConnection().GetMechanism().GetParameters()[kernel.NetNSURL] = newURL.String()
	}
	return next.Server(ctx).Request(ctx, request)
}

func (s *sendnsServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}
