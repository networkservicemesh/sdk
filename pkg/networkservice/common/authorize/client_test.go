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

func TestAuthorizeClientNoPolicy(t *testing.T) {
	client := authorize.NewClient(nil)
	conn, err := client.Request(context.Background(), &networkservice.NetworkServiceRequest{})
	assert.Nil(t, conn)
	assert.NotNil(t, err)
}

func TestAuthorizeClientSucceedingPolicy(t *testing.T) {
	client := authorize.NewClient(func(*peer.Peer, *networkservice.Connection) error {
		return nil
	})
	ctx := peer.NewContext(context.Background(), &peer.Peer{
		Addr: nil,
		AuthInfo: credentials.TLSInfo{
			State:          tls.ConnectionState{},
			CommonAuthInfo: credentials.CommonAuthInfo{},
		},
	})
	_, err := client.Request(ctx, &networkservice.NetworkServiceRequest{})
	assert.Nil(t, err)
}

func TestAuthorizeClientFailingPolicy(t *testing.T) {
	client := authorize.NewClient(func(*peer.Peer, *networkservice.Connection) error {
		return errors.New("Failed as expected")
	})
	ctx := peer.NewContext(context.Background(), &peer.Peer{
		Addr: nil,
		AuthInfo: credentials.TLSInfo{
			State:          tls.ConnectionState{},
			CommonAuthInfo: credentials.CommonAuthInfo{},
		},
	})
	_, err := client.Request(ctx, &networkservice.NetworkServiceRequest{})
	assert.NotNil(t, err)
}

func TestClientPropogatesError(t *testing.T) {
	logrus.SetOutput(ioutil.Discard)
	client := authorize.NewClient(func(*peer.Peer, *networkservice.Connection) error {
		return nil
	})
	client = checkerror.CheckPropagatesErrorClient(t, client)
	ctx := peer.NewContext(context.Background(), &peer.Peer{
		Addr: nil,
		AuthInfo: credentials.TLSInfo{
			State:          tls.ConnectionState{},
			CommonAuthInfo: credentials.CommonAuthInfo{},
		},
	})
	conn, err := client.Request(ctx, &networkservice.NetworkServiceRequest{})
	assert.NotNil(t, err)
	_, err = client.Close(ctx, conn)
	assert.NotNil(t, err)
}
