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

package recvfd

import (
	"context"
	"net/url"
	"os"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/edwarnicke/grpcfd"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type recvFDServer struct {
	fileMaps perConnectionFileMapMap
}

// NewServer - returns server chain element to recv FDs over the connection (if possible) for any Mechanism.Parameters[common.InodeURL]
// url of scheme 'inode'.
func NewServer() networkservice.NetworkServiceServer {
	return &recvFDServer{}
}

func (r *recvFDServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	// Get the grpcfd.FDRecver
	recv, ok := grpcfd.FromContext(ctx)
	if !ok {
		return next.Server(ctx).Request(ctx, request)
	}

	// Get the fileMap
	fileMap, _ := r.fileMaps.LoadOrStore(request.GetConnection().GetId(), &perConnectionFileMap{
		filesByInodeURL:    make(map[string]*os.File),
		inodeURLbyFilename: make(map[string]*url.URL),
	})

	// For each mechanism recv the FD and Swap the Inode for a file in InodeURL in Parameters
	for _, mechanism := range append(request.GetMechanismPreferences(), request.GetConnection().GetMechanism()) {
		err := recvFDAndSwapInodeToFile(ctx, fileMap, mechanism.GetParameters(), recv)
		if err != nil {
			return nil, err
		}
	}

	// Call the next server in the chain
	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		return nil, err
	}

	// Swap back from File to Inode in the InodeURL in the Parameters
	err = swapFileToInode(fileMap, conn.GetMechanism().GetParameters())
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func (r *recvFDServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	// Clean up the fileMap no matter what happens
	defer r.closeFiles(conn)

	// Get the grpcfd.FDRecver
	recv, ok := grpcfd.FromContext(ctx)
	if !ok {
		return next.Server(ctx).Close(ctx, conn)
	}

	// Get the fileMap
	fileMap, _ := r.fileMaps.LoadOrStore(conn.GetId(), &perConnectionFileMap{
		filesByInodeURL:    make(map[string]*os.File),
		inodeURLbyFilename: make(map[string]*url.URL),
	})

	// Recv the FD and Swap the Inode for a file in InodeURL in Parameters
	err := recvFDAndSwapInodeToFile(ctx, fileMap, conn.GetMechanism().GetParameters(), recv)
	if err != nil {
		return nil, err
	}

	// Call the next server in the chain
	_, err = next.Server(ctx).Close(ctx, conn)
	if err != nil {
		return nil, err
	}

	// Swap back from File to Inode in the InodeURL in the Parameters
	err = swapFileToInode(fileMap, conn.GetMechanism().GetParameters())
	return &empty.Empty{}, err
}

func (r *recvFDServer) closeFiles(conn *networkservice.Connection) {
	defer r.fileMaps.Delete(conn.GetId())

	fileMap, _ := r.fileMaps.LoadOrStore(conn.GetId(), &perConnectionFileMap{
		filesByInodeURL:    make(map[string]*os.File),
		inodeURLbyFilename: make(map[string]*url.URL),
	})

	for inodeURLStr, file := range fileMap.filesByInodeURL {
		delete(fileMap.filesByInodeURL, inodeURLStr)
		_ = file.Close()
	}
}
