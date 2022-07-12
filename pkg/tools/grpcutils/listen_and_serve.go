// Copyright (c) 2020 Cisco and/or its affiliates.
//
// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
//
// Copyright (c) 2022 Cisco and/or its affiliates.
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
	"fmt"
	"net"
	"net/url"
	"os"
	"path"

	"github.com/networkservicemesh/sdk/pkg/tools/log"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

const (
	unixScheme = "unix"
	tcpScheme  = "tcp"
)

// ListenAndServe listens on address with server.  Returns an chan err  which will
// receive an error and then be closed in the event that server.Serve(listener) returns an error.
func ListenAndServe(ctx context.Context, address *url.URL, server *grpc.Server) <-chan error {
	errCh := make(chan error, 1)

	// Create listener
	network, target := urlToNetworkTarget(address)

	if network == unixScheme {
		err := os.Remove(target)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			errCh <- errors.Wrap(err, "Cannot delete exist socket file")
			close(errCh)
			return errCh
		}
		basePath := path.Dir(target)
		if _, err = os.Stat(basePath); os.IsNotExist(err) {
			log.FromContext(ctx).Debugf("target folder %v not exists, Trying to create", basePath)
			if err = os.MkdirAll(basePath, os.ModePerm); err != nil {
				errCh <- errors.Wrapf(err, "Could not serve %v", target)
				close(errCh)
				return errCh
			}
		}
	}

	ln, err := net.Listen(network, target)

	if ln != nil {
		// We need to pass a real listener address into context, since we could specify random port.
		*address = *AddressToURL(ln.Addr())
	}

	if network == unixScheme {
		if _, err = os.Stat(target); err == nil {
			err = os.Chmod(target, os.ModePerm)
			if err != nil {
				errCh <- errors.Wrap(err, fmt.Sprintf("%v: сannot change mod", target))
			}
		}
	}

	// Serve
	go func() {
		if err != nil {
			errCh <- err
			close(errCh)
			return
		}
		defer func() {
			_ = ln.Close()
		}()

		// We need to montior context in separate goroutine to be able to stop server
		go func() {
			<-ctx.Done()
			server.Stop()
		}()

		err = server.Serve(ln)

		if err != nil {
			errCh <- err
		}
		close(errCh)
	}()
	return errCh
}

func urlToNetworkTarget(u *url.URL) (network, target string) {
	network = tcpScheme
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
