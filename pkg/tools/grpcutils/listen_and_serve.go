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

// Package grpcutils - provides a simple ListenAndServe for grpc
package grpcutils

import (
	"context"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"net"
	"net/url"

	"google.golang.org/grpc"
)

const (
	unixScheme = "unix"
)

// ListenAndServe listens on address with server.  Returns an chan err  which will
// receive an error and then be closed in the event that server.Serve(listener) returns an error.
func ListenAndServe(ctx context.Context, address *url.URL, server *grpc.Server) <-chan error {
	errCh := make(chan error, 1)

	// Serve
	go func() {
		// Create listener
		log.Entry(ctx).Println("creating listener...", address)
		network, target := urlToNetworkTarget(address)
		log.Entry(ctx).Println("urlToNetworkTarget:", network, target)
		ln, err := net.Listen(network, target)
		log.Entry(ctx).Println("listen:", ln, err)
		if err != nil {
			errCh <- err
			close(errCh)
			return
		}
		defer func() {
			log.Entry(ctx).Println("close:", ln, err)
			_ = ln.Close()
		}()
		log.Entry(ctx).Println("serve:")
		err = server.Serve(ln)
		log.Entry(ctx).Println("serve err:", err)
		select {
		case <-ctx.Done():
		default:
			errCh <- err
			close(errCh)
		}
	}()
	return errCh
}

func urlToNetworkTarget(u *url.URL) (network, target string) {
	network = "tcp"
	target = u.Host
	if u.Scheme == unixScheme {
		network = unixScheme
		target = u.Path
		if target == "" {
			target = u.Opaque
		}
	}
	return network, target
}
