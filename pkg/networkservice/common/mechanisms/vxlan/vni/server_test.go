// Copyright (c) 2021 Nordix Foundation.
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

package vni_test

import (
	"context"
	"net"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vxlan"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/vxlan/vni"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

func TestVNIServer(t *testing.T) {
	t.Parallel()
	t.Cleanup(func() { goleak.VerifyNone(t, goleak.IgnoreCurrent()) })
	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{},
		MechanismPreferences: []*networkservice.Mechanism{
			{
				Cls:  cls.REMOTE,
				Type: vxlan.MECHANISM,
			},
		},
	}
	request.GetConnection().Mechanism = &networkservice.Mechanism{
		Cls:  cls.REMOTE,
		Type: vxlan.MECHANISM,
	}
	vxlan.ToMechanism(request.GetConnection().GetMechanism()).SetSrcIP(net.ParseIP("192.0.2.1"))
	var port uint16 = 0
	server := next.NewNetworkServiceServer(
		metadata.NewServer(),
		vni.NewServer(net.ParseIP("192.0.2.2")),
	)
	conn, err := server.Request(context.Background(), request)
	assert.Nil(t, err)
	assert.NotNil(t, conn)
	mechanism := vxlan.ToMechanism(conn.GetMechanism())
	assert.NotNil(t, mechanism)
	assert.Equal(t, "192.0.2.2", mechanism.DstIP().String())
	assert.NotEqual(t, port, mechanism.DstPort())
	assert.NotEqual(t, 0, mechanism.VNI())

	port = 4466
	server = next.NewNetworkServiceServer(
		metadata.NewServer(),
		vni.NewServer(net.ParseIP("192.0.2.2"), vni.WithTunnelPort(port)),
	)
	conn, err = server.Request(context.Background(), request)
	assert.Nil(t, err)
	assert.NotNil(t, conn)
	mechanism = vxlan.ToMechanism(conn.GetMechanism())
	assert.NotNil(t, mechanism)
	assert.Equal(t, "192.0.2.2", mechanism.DstIP().String())
	assert.Equal(t, port, mechanism.DstPort())
	assert.NotEqual(t, 0, mechanism.VNI())
}

func TestVNIServerNonVxLAN(t *testing.T) {
	t.Parallel()
	t.Cleanup(func() { goleak.VerifyNone(t, goleak.IgnoreCurrent()) })
	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{},
		MechanismPreferences: []*networkservice.Mechanism{
			{
				Cls:  cls.LOCAL,
				Type: kernel.MECHANISM,
			},
		},
	}
	request.GetConnection().Mechanism = &networkservice.Mechanism{
		Cls:  cls.LOCAL,
		Type: kernel.MECHANISM,
	}
	server := next.NewNetworkServiceServer(vni.NewServer(net.ParseIP("192.0.2.2")))
	conn, err := server.Request(context.Background(), request)
	assert.Nil(t, err)
	assert.NotNil(t, conn)
	assert.Nil(t, vxlan.ToMechanism(conn.GetMechanism()))
}
