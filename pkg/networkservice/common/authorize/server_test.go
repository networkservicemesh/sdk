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

package authorize_test

import (
	"context"
	"crypto/tls"
	"io/ioutil"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/authorize"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/checks/checkerror"
)

func TestAuthorizeServer_Request_NoPolicy(t *testing.T) {
	server := authorize.NewServer(nil, nil)
	conn, err := server.Request(context.Background(), &networkservice.NetworkServiceRequest{})
	assert.Nil(t, conn)
	assert.NotNil(t, err)
}

func TestAuthorizeServer_Close_NoPolicy(t *testing.T) {
	server := authorize.NewServer(nil, nil)
	conn, err := server.Close(context.Background(), &networkservice.Connection{})
	assert.Nil(t, conn)
	assert.NotNil(t, err)
}

func TestAuthorizeServer_Request_SuccedingPolicy(t *testing.T) {
	server := authorize.NewServer(func(peer *peer.Peer, conn *networkservice.NetworkServiceRequest) error {
		return nil
	}, nil)
	ctx := peer.NewContext(context.Background(), &peer.Peer{
		Addr: nil,
		AuthInfo: credentials.TLSInfo{
			State:          tls.ConnectionState{},
			CommonAuthInfo: credentials.CommonAuthInfo{},
		},
	})
	_, err := server.Request(ctx, &networkservice.NetworkServiceRequest{})
	assert.Nil(t, err)
}
func TestAuthorizeServer_Close_SuccedingPolicy(t *testing.T) {
	server := authorize.NewServer(nil, func(peer *peer.Peer, conn *networkservice.Connection) error {
		return nil
	})
	ctx := peer.NewContext(context.Background(), &peer.Peer{
		Addr: nil,
		AuthInfo: credentials.TLSInfo{
			State:          tls.ConnectionState{},
			CommonAuthInfo: credentials.CommonAuthInfo{},
		},
	})
	_, err := server.Close(ctx, &networkservice.Connection{})
	assert.Nil(t, err)
}

func TestAuthorizeServer_Request_FailingPolicy(t *testing.T) {
	server := authorize.NewServer(func(peer *peer.Peer, conn *networkservice.NetworkServiceRequest) error {
		return errors.New("Failed as expected")
	}, nil)
	ctx := peer.NewContext(context.Background(), &peer.Peer{
		Addr: nil,
		AuthInfo: credentials.TLSInfo{
			State:          tls.ConnectionState{},
			CommonAuthInfo: credentials.CommonAuthInfo{},
		},
	})
	_, err := server.Request(ctx, &networkservice.NetworkServiceRequest{})
	assert.NotNil(t, err)
}
func TestAuthorizeServer_Close_FailingPolicy(t *testing.T) {
	server := authorize.NewServer(nil, func(peer *peer.Peer, conn *networkservice.Connection) error {
		return errors.New("Failed as expected")
	})
	ctx := peer.NewContext(context.Background(), &peer.Peer{
		Addr: nil,
		AuthInfo: credentials.TLSInfo{
			State:          tls.ConnectionState{},
			CommonAuthInfo: credentials.CommonAuthInfo{},
		},
	})
	_, err := server.Close(ctx, &networkservice.Connection{})
	assert.NotNil(t, err)
}

func TestServerPropogatesError(t *testing.T) {
	logrus.SetOutput(ioutil.Discard)
	server := authorize.NewServer(func(peer *peer.Peer, conn *networkservice.NetworkServiceRequest) error {
		return nil
	}, func(*peer.Peer, *networkservice.Connection) error {
		return nil
	})
	server = checkerror.CheckPropogatesErrorServer(t, server)
	ctx := peer.NewContext(context.Background(), &peer.Peer{
		Addr: nil,
		AuthInfo: credentials.TLSInfo{
			State:          tls.ConnectionState{},
			CommonAuthInfo: credentials.CommonAuthInfo{},
		},
	})
	conn, err := server.Request(ctx, &networkservice.NetworkServiceRequest{})
	assert.NotNil(t, err)
	_, err = server.Close(ctx, conn)
	assert.NotNil(t, err)
}
