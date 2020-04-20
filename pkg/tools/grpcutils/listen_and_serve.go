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
	"net"
	"net/url"

	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type contextKeyType string

const (
	unixScheme                = "unix"
	errorKey   contextKeyType = "Error"
)

// Err - return the error (if any) stored by ListenAndServe in the context
func Err(ctx context.Context) error {
	if value := ctx.Value(errorKey); value != nil {
		return *value.(*error)
	}
	return nil
}

// ListenAndServe listens on address with server.  Returns a context which will be canceled in the event that
// server.Serve(listener) returns an error.  The resulting error can then be retrieve from the returned context with.
// grpcutils.Err(ctx)
func ListenAndServe(ctx context.Context, address *url.URL, server *grpc.Server) context.Context {
	ctx, cancel := context.WithCancel(ctx)
	var listenAndServeError error
	ctx = context.WithValue(ctx, errorKey, &listenAndServeError)

	// Serve
	go func() {
		// Create listener
		network, target := urlToNetworkTarget(address)
		ln, err := net.Listen(network, target)
		if err != nil {
			log.Entry(ctx).Fatalf("failed to listen on %q: %v", address, err)
		}
		defer func() {
			_ = ln.Close()
		}()
		err = server.Serve(ln)
		if err != nil {
			select {
			case <-ctx.Done():
			default:
				listenAndServeError = err
				cancel()
			}
		}
	}()
	return ctx
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
