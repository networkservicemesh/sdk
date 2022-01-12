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

// Package sendfd provides a registry.NetworkServiceEndpointRegistryClient chain element to convert any unix file socket
// endpoint.URLs into 'inode://${dev}/${ino}' urls and send the fd over the unix file socket.
package sendfd

import (
	"context"
	"net/url"

	"github.com/edwarnicke/grpcfd"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

type sendFDNSEClient struct{}

// NewNetworkServiceEndpointRegistryClient - creates a Client that if endpoint.Url is of scheme "unix" will replace it with an "inode" scheme url and send the FD over the unix socket
func NewNetworkServiceEndpointRegistryClient() registry.NetworkServiceEndpointRegistryClient {
	return &sendFDNSEClient{}
}

func (s *sendFDNSEClient) Register(ctx context.Context, endpoint *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	// Get the grpcfd.FDSender
	rpcCredentials := grpcfd.PerRPCCredentials(grpcfd.PerRPCCredentialsFromCallOptions(opts...))
	opts = append(opts, grpc.PerRPCCredentials(rpcCredentials))
	sender, _ := grpcfd.FromPerRPCCredentials(rpcCredentials)

	inodeURLToUnixURLMap := make(map[string]string)

	if err := sendFDAndSwapFileToInode(sender, endpoint, inodeURLToUnixURLMap); err != nil {
		return nil, err
	}

	returnedEndpoint, err := next.NetworkServiceEndpointRegistryClient(ctx).Register(ctx, endpoint, opts...)
	if err != nil {
		return nil, err
	}

	// Translate the InodeURl mechanism *back to a proper file://${path} url
	if fileURLStr, ok := inodeURLToUnixURLMap[returnedEndpoint.Url]; ok {
		returnedEndpoint.Url = fileURLStr
	}
	return returnedEndpoint, nil
}

func (s *sendFDNSEClient) Find(ctx context.Context, query *registry.NetworkServiceEndpointQuery, opts ...grpc.CallOption) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	return next.NetworkServiceEndpointRegistryClient(ctx).Find(ctx, query, opts...)
}

func (s *sendFDNSEClient) Unregister(ctx context.Context, endpoint *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*empty.Empty, error) {
	// Get the grpcfd.FDSender
	rpcCredentials := grpcfd.PerRPCCredentials(grpcfd.PerRPCCredentialsFromCallOptions(opts...))
	opts = append(opts, grpc.PerRPCCredentials(rpcCredentials))
	sender, _ := grpcfd.FromPerRPCCredentials(rpcCredentials)

	inodeURLToFileURLMap := make(map[string]string)

	if err := sendFDAndSwapFileToInode(sender, endpoint, inodeURLToFileURLMap); err != nil {
		return nil, err
	}

	_, err := next.NetworkServiceEndpointRegistryClient(ctx).Unregister(ctx, endpoint, opts...)
	if err != nil {
		return nil, err
	}

	// Translate the InodeURl mechanism *back to a proper file://${path} url
	if fileURLStr, ok := inodeURLToFileURLMap[endpoint.Url]; ok {
		endpoint.Url = fileURLStr
	}
	return &empty.Empty{}, nil
}

func sendFDAndSwapFileToInode(sender grpcfd.FDSender, endpoint *registry.NetworkServiceEndpoint, inodeURLToUnixURLMap map[string]string) error {
	// Transform string to URL for correctness checking and ease of use
	unixURL, err := url.Parse(endpoint.Url)
	if err != nil {
		return errors.WithStack(err)
	}

	// Is it a file?
	if unixURL.Scheme != "unix" {
		return nil
	}
	// Translate the file to an inodeURL of the form inode://${dev}/${ino}
	inodeURL, err := grpcfd.FilenameToURL(unixURL.Path)
	if err != nil {
		return errors.WithStack(err)
	}
	// Send the file
	errCh := sender.SendFilename(unixURL.Path)
	select {
	case err := <-errCh:
		// If we immediately return an error... the file probably doesn't exist
		return errors.WithStack(err)
	default:
		// But don't wait for any subsequent errors... they won't arrive till after we've sent the GRPC message
	}
	// swap the InodeURL parameter for a real inode://${dev}/${ino} URL
	endpoint.Url = inodeURL.String()
	// remember the original fileURL so we can translate back later
	inodeURLToUnixURLMap[inodeURL.String()] = unixURL.String()
	return nil
}
