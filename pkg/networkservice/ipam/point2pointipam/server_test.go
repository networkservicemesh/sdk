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

package point2pointipam_test

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/networkservice/ipam/point2pointipam"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/assert"
)

func newRequest() *networkservice.NetworkServiceRequest {
	return &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id:             "id",
			NetworkService: "ns",
			Context: &networkservice.ConnectionContext{
				IpContext: &networkservice.IPContext{},
			},
		},
		MechanismPreferences: nil,
	}
}

func TestServer(t *testing.T) {
	_, ipnet, err := net.ParseCIDR("192.168.3.4/16")
	require.NoError(t, err)
	srv := point2pointipam.NewServer(ipnet)

	conn1, err := srv.Request(context.Background(), newRequest())
	assert.NoError(t, err)

	assert.Equal(t, "192.168.0.0/32", conn1.Context.IpContext.DstIpAddr)
	assert.Equal(t, "192.168.0.1/32", conn1.Context.IpContext.SrcIpAddr)

	conn2, err := srv.Request(context.Background(), newRequest())
	assert.NoError(t, err)

	assert.Equal(t, "192.168.0.2/32", conn2.Context.IpContext.DstIpAddr)
	assert.Equal(t, "192.168.0.3/32", conn2.Context.IpContext.SrcIpAddr)

	_, err = srv.Close(context.Background(), conn1)
	assert.NoError(t, err)

	conn3, err := srv.Request(context.Background(), newRequest())
	assert.NoError(t, err)

	assert.Equal(t, "192.168.0.0/32", conn3.Context.IpContext.DstIpAddr)
	assert.Equal(t, "192.168.0.1/32", conn3.Context.IpContext.SrcIpAddr)

	conn4, err := srv.Request(context.Background(), newRequest())
	assert.NoError(t, err)

	assert.Equal(t, "192.168.0.4/32", conn4.Context.IpContext.DstIpAddr)
	assert.Equal(t, "192.168.0.5/32", conn4.Context.IpContext.SrcIpAddr)
}

func TestNilPrefixes(t *testing.T) {
	srv := point2pointipam.NewServer()
	_, err := srv.Request(context.Background(), newRequest())
	require.Error(t, err)
	_, cidr1, _ := net.ParseCIDR("192.168.0.1/32")

	srv = point2pointipam.NewServer(
		nil,
		cidr1,
		nil,
	)
	_, err = srv.Request(context.Background(), newRequest())

	assert.Error(t, err)
}

func TestExclude32Prefix(t *testing.T) {
	_, ipnet, err := net.ParseCIDR("192.168.1.0/24")
	assert.Nil(t, err)
	srv := point2pointipam.NewServer(ipnet)
	assert.NotNil(t, srv)
	assert.NoError(t, err)

	// Test center of assigned
	req1 := newRequest()
	req1.Connection.Context.IpContext.ExcludedPrefixes = []string{"192.168.1.1/32", "192.168.1.3/32", "192.168.1.8/32"}
	conn1, err := srv.Request(context.Background(), req1)
	assert.NoError(t, err)
	assert.Equal(t, "192.168.1.0/32", conn1.Context.IpContext.DstIpAddr)
	assert.Equal(t, "192.168.1.2/32", conn1.Context.IpContext.SrcIpAddr)

	// Test exclude before assigned
	req2 := newRequest()
	req2.Connection.Context.IpContext.ExcludedPrefixes = []string{"192.168.1.1/32", "192.168.1.3/32", "192.168.1.8/32"}
	conn2, err := srv.Request(context.Background(), req2)
	assert.NoError(t, err)
	assert.Equal(t, "192.168.1.4/32", conn2.Context.IpContext.DstIpAddr)
	assert.Equal(t, "192.168.1.5/32", conn2.Context.IpContext.SrcIpAddr)

	// Test after assigned
	req3 := newRequest()
	req3.Connection.Context.IpContext.ExcludedPrefixes = []string{"192.168.1.1/32", "192.168.1.3/32", "192.168.1.8/32"}
	conn3, err := srv.Request(context.Background(), req3)
	assert.NoError(t, err)
	assert.Equal(t, "192.168.1.6/32", conn3.Context.IpContext.DstIpAddr)
	assert.Equal(t, "192.168.1.7/32", conn3.Context.IpContext.SrcIpAddr)
}

func TestOutOfIPs(t *testing.T) {
	_, ipnet, err := net.ParseCIDR("192.168.1.2/31")
	assert.NoError(t, err)
	srv := point2pointipam.NewServer(ipnet)
	assert.NotNil(t, srv)

	req1 := newRequest()
	conn1, err := srv.Request(context.Background(), req1)
	assert.NoError(t, err)
	assert.Equal(t, "192.168.1.2/32", conn1.Context.IpContext.DstIpAddr)
	assert.Equal(t, "192.168.1.3/32", conn1.Context.IpContext.SrcIpAddr)

	req2 := newRequest()
	conn2, err := srv.Request(context.Background(), req2)
	assert.Nil(t, conn2)
	assert.Error(t, err)
}

func TestAllIPsExcluded(t *testing.T) {
	_, ipnet, err := net.ParseCIDR("192.168.1.2/31")
	assert.NoError(t, err)
	srv := point2pointipam.NewServer(ipnet)
	assert.NotNil(t, srv)

	req1 := newRequest()
	req1.Connection.Context.IpContext.ExcludedPrefixes = []string{
		"192.168.1.2/31",
	}
	conn1, err := srv.Request(context.Background(), req1)
	assert.Nil(t, conn1)
	assert.Error(t, err)
}
