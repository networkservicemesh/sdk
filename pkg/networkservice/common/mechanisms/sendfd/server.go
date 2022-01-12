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

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type sendFDServer struct{}

// NewServer - returns server which sends any "file://" Mechanism.Parameters[common.InodeURLs]s across the connection as fds (if possible) to the client
func NewServer() networkservice.NetworkServiceServer {
	return &sendFDServer{}
}

func (s sendFDServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	// Call the next server chain element in the chain
	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		return nil, err
	}

	// Get the grpcfd.FDSender
	sender, ok := grpcfd.FromContext(ctx)
	if !ok {
		return conn, nil
	}

	// Send the FD and swap the FileURL for an InodeURL
	inodeURLToFileURLMap := make(map[string]string)
	if err := sendFDAndSwapFileToInode(sender, conn.GetMechanism().GetParameters(), inodeURLToFileURLMap); err != nil {
		return nil, err
	}
	return conn, nil
}

func (s sendFDServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	// Call the next server chain element in the chain
	_, err := next.Server(ctx).Close(ctx, conn)
	if err != nil {
		return nil, err
	}

	sender, ok := grpcfd.FromContext(ctx)
	if !ok {
		return &empty.Empty{}, nil
	}

	// Send the FD and swap the FileURL for an InodeURL
	inodeURLToFileURLMap := make(map[string]string)
	if err := sendFDAndSwapFileToInode(sender, conn.GetMechanism().GetParameters(), inodeURLToFileURLMap); err != nil {
		return nil, err
	}

	return &empty.Empty{}, nil
}
