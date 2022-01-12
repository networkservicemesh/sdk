// Copyright (c) 2020-2022 Cisco and/or its affiliates.
//
// Copyright (c) 2021-2022 Doc.ai and/or its affiliates.
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

package recvfd

import (
	"context"
	"net/url"
	"os"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"google.golang.org/grpc"

	"github.com/edwarnicke/grpcfd"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type recvFDClient struct {
	fileMaps perConnectionFileMapMap
}

// NewClient - returns client chain element to recv FDs over the connection (if possible) for any Mechanism.Parameters[common.InodeURL]
// url of scheme 'inode'.
func NewClient() networkservice.NetworkServiceClient {
	return &recvFDClient{}
}

func (r *recvFDClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	// Get the grpcfd.FDRecver
	rpcCredentials := grpcfd.PerRPCCredentials(grpcfd.PerRPCCredentialsFromCallOptions(opts...))
	opts = append(opts, grpc.PerRPCCredentials(rpcCredentials))
	recv, _ := grpcfd.FromPerRPCCredentials(rpcCredentials)

	// Call the next Client in the chain
	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return nil, err
	}

	// Get the fileMap
	fileMap, _ := r.fileMaps.LoadOrStore(conn.GetId(), &perConnectionFileMap{
		filesByInodeURL:    make(map[string]*os.File),
		inodeURLbyFilename: make(map[string]*url.URL),
	})

	// Recv the FD and swap theInode to File in the Parameters for the returned connection mechanism
	err = recvFDAndSwapInodeToFile(ctx, fileMap, conn.GetMechanism().GetParameters(), recv)
	if err != nil {
		return nil, err
	}

	// Return connection
	return conn, nil
}

func (r *recvFDClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	// Get the grpcfd.FDRecver
	rpcCredentials := grpcfd.PerRPCCredentials(grpcfd.PerRPCCredentialsFromCallOptions(opts...))
	opts = append(opts, grpc.PerRPCCredentials(rpcCredentials))
	recv, _ := grpcfd.FromPerRPCCredentials(rpcCredentials)

	// Get the fileMap
	fileMap, _ := r.fileMaps.LoadOrStore(conn.GetId(), &perConnectionFileMap{
		filesByInodeURL:    make(map[string]*os.File),
		inodeURLbyFilename: make(map[string]*url.URL),
	})

	go func(fileMap *perConnectionFileMap, conn *networkservice.Connection) {
		<-ctx.Done()
		for i, file := range fileMap.filesByInodeURL {
			log.FromContext(ctx).Debugf("Closing file %q (%q) related to closed connection %s", file.Name(), i, conn.GetId())
			_ = file.Close()
		}
	}(fileMap, conn)

	// Whatever happens, clean up the fileMap
	defer r.fileMaps.Delete(conn.GetId())

	// Call the next Client in the chain
	_, err := next.Client(ctx).Close(ctx, conn, opts...)
	if err != nil {
		return nil, err
	}

	// Recv the FD and swap theInode to File in the Parameters for the returned connection mechanism
	err = recvFDAndSwapInodeToFile(ctx, fileMap, conn.GetMechanism().GetParameters(), recv)
	if err != nil {
		return nil, err
	}

	return &empty.Empty{}, nil
}
