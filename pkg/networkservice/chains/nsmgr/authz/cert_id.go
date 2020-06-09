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

// Package authz has some identification routines used by callback of nsmgr
package authz

import (
	"context"
	"errors"
	"net/url"

	"google.golang.org/grpc/metadata"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
)

// CertificateURL - return URL from certificate
func CertificateURL(ctx context.Context) (*url.URL, error) {
	p, ok := peer.FromContext(ctx)
	if !ok {
		err := errors.New("no peer is provided")
		logrus.Error(err)
		return nil, err
	}
	tlsInfo, tlsOk := p.AuthInfo.(credentials.TLSInfo)
	if !tlsOk {
		err := errors.New("no TLS info is provided")
		logrus.Error(err)
		return nil, err
	}
	if len(tlsInfo.State.PeerCertificates) == 0 {
		err := errors.New("no TLS peer certificate info is provided")
		logrus.Error(err)
		return nil, err
	}
	uris := tlsInfo.State.PeerCertificates[0].URIs
	if len(uris) == 0 {
		err := errors.New("no TLS peer certificate info is provided")
		logrus.Error(err)
		return nil, err
	}
	return uris[0], nil
}

// IdentityByEndpointID - return identity by :endpoint-id
func IdentityByEndpointID(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		err := errors.New("no metadata provided")
		logrus.Error(err)
		return "", err
	}
	return md.Get("endpoint-id")[0], nil
}

// WithCallbackEndpointID - pass with :endpoint-id a correct endpoint identity.
func WithCallbackEndpointID(ctx context.Context, endpoint *url.URL) context.Context {
	return metadata.AppendToOutgoingContext(ctx, "endpoint-id", endpoint.Path)
}
