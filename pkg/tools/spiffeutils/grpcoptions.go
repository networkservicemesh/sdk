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

package spiffeutils

import (
	"context"
	"time"

	"github.com/spiffe/go-spiffe/spiffe"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	// DefaultTimeout - default for use as timeout in spiffe grpc options
	DefaultTimeout = time.Second
)

// WithSpiffe - returns a grpc.DialOption for using Spiffe IDs if available, grpc.WithInsecure() otherwise
//              peer           - *spiffe.TLSPeer - spiffe TLS Peer
//              timeout        - time.Duration to allow maximally to contacting spiffe agent before giving up and
//                               returning grpc.EmptyDialOption
//              Example:
//                 tlsPeer, _ := spiffeutils.NewTLSPeer(spiffe.WithWorkloadAPIAddr(agentUrl),spiffe.WithLogger(logrus.StandardLogger()))
//                 conn, err := grpc.DialContext(ctx,target),grpcoptions.WithSpiffe(tlsPeer,grpcoptions.DefaultTimeout))
func WithSpiffe(peer TLSPeer, timeout time.Duration) grpc.DialOption {
	if peer == nil {
		return grpc.WithInsecure()
	}
	ctx, cancelFunc := context.WithTimeout(context.Background(), timeout)
	defer cancelFunc()
	tlsConfig, err := peer.GetConfig(ctx, spiffe.ExpectAnyPeer())
	if err != nil {
		return grpc.WithInsecure()
	}
	tlsConfig.InsecureSkipVerify = true
	return grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig))
}

// SpiffeCreds - returns a grpc.ServerOption for using Spiffe IDs if available, grpc.EmptyServerOption otherwise
//              peer           - *spiffe.TLSPeer - spiffe TLS Peer
//              timeout        - time.Duration to allow maximally to contacting spiffe agent before giving up and
//                               returning grpc.EmptyServerOption
//              Example:
//                 tlsPeer, _ := spiffeutils.NewTLSPeer(spiffe.WithWorkloadAPIAddr(agentUrl),spiffe.WithLogger(logrus.StandardLogger()))
//                 server := grpc.NewServer(grpcoptions.SpiffeCreds(tlsPeer, grpcoptions.DefaultTimeout))
func SpiffeCreds(peer TLSPeer, timeout time.Duration) grpc.ServerOption {
	if peer == nil {
		return grpc.EmptyServerOption{}
	}
	ctx, cancelFunc := context.WithTimeout(context.Background(), timeout)
	defer cancelFunc()
	tlsConfig, err := peer.GetConfig(ctx, spiffe.ExpectAnyPeer())
	if err != nil {
		return grpc.EmptyServerOption{}
	}
	tlsConfig.InsecureSkipVerify = true
	return grpc.Creds(credentials.NewTLS(tlsConfig))
}
