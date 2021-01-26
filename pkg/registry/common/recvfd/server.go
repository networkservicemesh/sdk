// Copyright (c) 2020-2021 Cisco and/or its affiliates.
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

// +build linux

// Package recvfd provides an NSE registry server chain element that:
//  1. Receives and fd over a unix file socket if the nse.URL is an inode://${dev}/${inode} url
//  2. Rewrites the nse.URL to unix:///proc/${pid}/fd/${fd} so it can be used by a normal dialer
package recvfd

import (
	"context"
	"net/url"
	"os"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"

	"github.com/edwarnicke/grpcfd"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

type recvfdNseServer struct {
	fileMaps perEndpointFileMapMap
}

// NewNetworkServiceEndpointRegistryServer - creates new NSE registry chain element that will:
//  1. Receive and fd over a unix file socket if the nse.URL is an inode://${dev}/${inode} url
//  2. Rewrite the nse.URL to unix:///proc/${pid}/fd/${fd} so it can be used by a normal dialer
func NewNetworkServiceEndpointRegistryServer() registry.NetworkServiceEndpointRegistryServer {
	return &recvfdNseServer{}
}

func (r *recvfdNseServer) Register(ctx context.Context, endpoint *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	recv, ok := grpcfd.FromContext(ctx)
	if !ok {
		return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, endpoint)
	}

	// Get the fileMap
	fileMap := &perEndpointFileMap{
		filesByInodeURL:    make(map[string]*os.File),
		inodeURLbyFilename: make(map[string]*url.URL),
	}
	endpointName := endpoint.Name
	// If name is specified, let's use it, since it could be heal/update request
	if endpointName != "" {
		fileMap, _ = r.fileMaps.LoadOrStore(endpoint.GetName(), fileMap)
	}

	// Recv the FD and Swap the Inode for a file in InodeURL in Parameters
	endpoint = endpoint.Clone()
	err := recvFDAndSwapInodeToUnix(ctx, fileMap, endpoint, recv)
	if err != nil {
		return nil, err
	}

	// Call the next server in the chain
	returnedEndpoint, err := next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, endpoint)
	if err != nil {
		return nil, err
	}
	returnedEndpoint = returnedEndpoint.Clone()

	if endpointName != returnedEndpoint.Name {
		// We need to store new value
		r.fileMaps.Store(endpoint.GetName(), fileMap)
	}
	// Swap back from File to Inode in the InodeURL in the Parameters
	err = swapFileToInode(fileMap, returnedEndpoint, false)
	if err != nil {
		return nil, err
	}
	return returnedEndpoint, nil
}

func (r *recvfdNseServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, server)
}

func (r *recvfdNseServer) Unregister(ctx context.Context, endpoint *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	if endpoint.GetName() == "" {
		return nil, errors.New("invalid endpoint specified")
	}
	// Get the grpcfd.FDRecver
	recv, ok := grpcfd.FromContext(ctx)
	if !ok {
		return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, endpoint)
	}
	// Get the fileMap
	fileMap, _ := r.fileMaps.LoadOrStore(endpoint.GetName(), &perEndpointFileMap{
		filesByInodeURL:    make(map[string]*os.File),
		inodeURLbyFilename: make(map[string]*url.URL),
	})
	// Clean up the fileMap no matter what happens
	defer r.fileMaps.Delete(endpoint.GetName())

	// Recv the FD and Swap the Inode for a file in InodeURL in Parameters
	endpoint = endpoint.Clone()
	err := recvFDAndSwapInodeToUnix(ctx, fileMap, endpoint, recv)
	if err != nil {
		return nil, err
	}

	// Call the next server in the chain
	_, err = next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, endpoint)
	if err != nil {
		return nil, err
	}

	// Swap back from File to Inode in the InodeURL in the Parameters
	endpoint = endpoint.Clone()
	err = swapFileToInode(fileMap, endpoint, true)
	if err != nil {
		return nil, err
	}
	return &empty.Empty{}, nil
}

func recvFDAndSwapInodeToUnix(ctx context.Context, fileMap *perEndpointFileMap, endpoint *registry.NetworkServiceEndpoint, recv grpcfd.FDRecver) error {
	inodeURL, err := url.Parse(endpoint.GetUrl())
	if err != nil {
		return errors.WithStack(err)
	}

	// Is it an inode?
	if inodeURL.Scheme != "inode" {
		return nil
	}

	<-fileMap.executor.AsyncExec(func() {
		file, ok := fileMap.filesByInodeURL[endpoint.GetUrl()]
		if !ok {
			// If we don't have a file for that inode... get it from the grpcfd.FDRecver
			var fileCh <-chan *os.File
			fileCh, err = recv.RecvFileByURL(endpoint.GetUrl())
			if err != nil {
				err = errors.WithStack(err)
				return
			}
			// Wait for the file to arrive on the fileCh or the context to expire
			select {
			case <-ctx.Done():
				err = ctx.Err()
				return
			case file = <-fileCh:
				// If we get the file, remember it in the fileMap so we can reuse it later
				// Note: This is done because we want to present a single consistent filename to
				// any of the other chain elements using the information, and since that filename will be
				// file:///proc/${pid}/fd/${fd} we need to remember it because each time we get it from the
				// grpcfd.Recver it will be a *different* fd and thus a different filename
				fileMap.filesByInodeURL[inodeURL.String()] = file
			}
		}
		// Swap out the inodeURL for a fileURL in the parameters
		unixURL := &url.URL{Scheme: "unix", Path: file.Name()}
		endpoint.Url = unixURL.String()

		// Remember the swap so we can undo it later
		fileMap.inodeURLbyFilename[file.Name()] = inodeURL
	})
	return err
}

func swapFileToInode(fileMap *perEndpointFileMap, endpoint *registry.NetworkServiceEndpoint, closeAllFiles bool) error {
	// Transform string to URL for correctness checking and ease of use
	unixURL, err := url.Parse(endpoint.GetUrl())
	if err != nil {
		return errors.WithStack(err)
	}

	// Is it a file?
	if unixURL.Scheme != "unix" {
		return nil
	}
	<-fileMap.executor.AsyncExec(func() {
		// Do we have an inodeURL to translate it back to?
		inodeURL, ok := fileMap.inodeURLbyFilename[unixURL.Path]
		if !ok {
			return
		}
		// Swap the fileURL for the inodeURL in parameters
		endpoint.Url = inodeURL.String()

		// If closeAllFiles == true, close any files we may have open for any other inodes
		// This is used to clean up files sent by MechanismPreferences that were *not* selected to be the
		// connection mechanism
		for inodeURLStr, file := range fileMap.filesByInodeURL {
			if closeAllFiles || inodeURLStr != inodeURL.String() {
				delete(fileMap.filesByInodeURL, inodeURLStr)
				_ = file.Close()
			}
		}
	})
	return nil
}
