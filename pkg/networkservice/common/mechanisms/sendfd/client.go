// Copyright (c) 2020-2022 Cisco and/or its affiliates.
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

//go:build linux
// +build linux

package sendfd

import (
	"context"

	"github.com/edwarnicke/grpcfd"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/common"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type sendFDClient struct{}

// NewClient - returns client which sends any "file://" Mechanism.Parameters[common.InodeURLs]s across the connection as fds (if possible) to the server
func NewClient() networkservice.NetworkServiceClient {
	return &sendFDClient{}
}

func (s *sendFDClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	// Get the grpcfd.FDSender
	rpcCredentials := grpcfd.PerRPCCredentials(grpcfd.PerRPCCredentialsFromCallOptions(opts...))
	opts = append(opts, grpc.PerRPCCredentials(rpcCredentials))
	sender, _ := grpcfd.FromPerRPCCredentials(rpcCredentials)

	// Iterate over mechanisms
	inodeURLToFileURLMap := make(map[string]string)
	for _, mechanism := range append(request.GetMechanismPreferences(), request.GetConnection().GetMechanism()) {
		if err := sendFDAndSwapFileToInode(sender, mechanism.GetParameters(), inodeURLToFileURLMap); err != nil {
			return nil, err
		}
	}
	// Call the next Client in the chain
	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return nil, err
	}
	// Is we don't have a InodeURL Parameter on the selected Mechanism... we don't need to translate it back
	if conn.GetMechanism().GetParameters() == nil || conn.GetMechanism().GetParameters()[common.InodeURL] == "" {
		return conn, nil
	}
	// Translate the InodeURl mechanism *back to a proper file://${path} url
	if fileURLStr, ok := inodeURLToFileURLMap[conn.GetMechanism().GetParameters()[common.InodeURL]]; ok {
		conn.GetMechanism().GetParameters()[common.InodeURL] = fileURLStr
	}
	return conn, nil
}

func (s *sendFDClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	rpcCredentials := grpcfd.PerRPCCredentials(grpcfd.PerRPCCredentialsFromCallOptions(opts...))
	opts = append(opts, grpc.PerRPCCredentials(rpcCredentials))
	sender, _ := grpcfd.FromPerRPCCredentials(rpcCredentials)

	// Send the FD and swap the FileURL for an InodeURL
	inodeURLToFileURLMap := make(map[string]string)
	if err := sendFDAndSwapFileToInode(sender, conn.GetMechanism().GetParameters(), inodeURLToFileURLMap); err != nil {
		return nil, err
	}

	// Call the next Client in the chain
	_, err := next.Client(ctx).Close(ctx, conn, opts...)
	if err != nil {
		return nil, err
	}

	// Translate the InodeURl mechanism *back to a proper file://${path} url
	if fileURLStr, ok := inodeURLToFileURLMap[conn.GetMechanism().GetParameters()[common.InodeURL]]; ok {
		conn.GetMechanism().GetParameters()[common.InodeURL] = fileURLStr
	}
	return &empty.Empty{}, nil
}
