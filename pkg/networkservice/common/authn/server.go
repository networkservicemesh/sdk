// Copyright (c) 2020 Doc.ai and/or its affiliates.
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

// Package authn provides chain elements for authentication
package authn

import (
	"context"
	"crypto/x509"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/security"
)

type authnServer struct {
	p security.Provider
}

// NewServer - returns a new authentication server
//			   p - provider of TLS certificate
func NewServer(p security.Provider) networkservice.NetworkServiceServer {
	return &authnServer{
		p: p,
	}
}

func (a *authnServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	x509crt, err := peerCertificateFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unable to get peer's TLS cert from context: %v", err)
	}

	token := request.GetConnection().GetPath().GetPathSegments()[request.GetConnection().GetPath().GetIndex()].GetToken()
	if err := security.VerifyToken(token, x509crt); err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unable to verify token: %v", err)
	}

	return next.Server(ctx).Request(ctx, request)
}

func (a *authnServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}

func peerCertificateFromContext(ctx context.Context) (*x509.Certificate, error) {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return nil, errors.New("missing peer TLSCred")
	}

	tlsInfo, ok := p.AuthInfo.(credentials.TLSInfo)
	if !ok {
		return nil, errors.New("peer has wrong type")
	}

	if len(tlsInfo.State.PeerCertificates) == 0 {
		return nil, errors.New("peer's certificate list is empty")
	}

	return tlsInfo.State.PeerCertificates[0], nil
}
