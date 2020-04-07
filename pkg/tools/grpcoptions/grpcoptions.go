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

// Package grpcoptions -
package grpcoptions

import (
	"context"
	"net/url"
	"time"

	"github.com/spiffe/go-spiffe/spiffe"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// WithSpiffe - returns a grpc.DialOption for using Spiffe IDs if available, grpc.EmptyDialOption otherwise
//              spiffeAgentURL - *url.URL for the spiffe agent
//              timeout        - time.Duration to allow maximally to contacting spiffe agent before giving up and
//                               returning grpc.EmptyDialOption
func WithSpiffe(spiffeAgentURL *url.URL, timeout time.Duration) grpc.DialOption {
	peer, err := spiffe.NewTLSPeer(spiffe.WithWorkloadAPIAddr(spiffeAgentURL.String()))
	if err != nil {
		return grpc.WithInsecure()
	}
	ctx, cancelFunc := context.WithTimeout(context.Background(), timeout)
	defer cancelFunc()
	tlsConfig, err := peer.GetConfig(ctx, spiffe.ExpectAnyPeer())
	if err != nil {
		return grpc.WithInsecure()
	}
	return grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig))
}

// SpiffeCreds - returns a grpc.ServerOption for using Spiffe IDs if available, grpc.EmptyServerOption otherwise
//              timeout        - time.Duration to allow maximally to contacting spiffe agent before giving up and
//                               returning grpc.EmptyServerOption
func SpiffeCreds(spiffeAgentURL *url.URL, timeout time.Duration) grpc.ServerOption {
	peer, err := spiffe.NewTLSPeer(spiffe.WithWorkloadAPIAddr(spiffeAgentURL.String()))
	if err != nil {
		return grpc.EmptyServerOption{}
	}
	ctx, cancelFunc := context.WithTimeout(context.Background(), timeout)
	defer cancelFunc()
	tlsConfig, err := peer.GetConfig(ctx, spiffe.ExpectAnyPeer())
	if err != nil {
		return grpc.EmptyServerOption{}
	}
	return grpc.Creds(credentials.NewTLS(tlsConfig))
}
