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
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vxlan"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/vxlan/vni"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

func TestVNIClient(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{},
		MechanismPreferences: []*networkservice.Mechanism{
			{
				Cls:  cls.REMOTE,
				Type: vxlan.MECHANISM,
			},
		},
	}
	var port uint16 = 0
	c := next.NewNetworkServiceClient(vni.NewClient(net.ParseIP("192.0.2.1")))
	conn, err := c.Request(context.Background(), request)
	assert.Nil(t, err)
	assert.NotNil(t, conn)
	mechanism := vxlan.ToMechanism(request.GetMechanismPreferences()[0])
	assert.NotNil(t, mechanism)
	assert.Equal(t, "192.0.2.1", mechanism.SrcIP().String())
	assert.NotEqual(t, port, mechanism.SrcPort())

	port = 4466
	c = next.NewNetworkServiceClient(vni.NewClient(net.ParseIP("192.0.2.1"), vni.WithTunnelPort(port)))
	conn, err = c.Request(context.Background(), request)
	assert.Nil(t, err)
	assert.NotNil(t, conn)
	mechanism = vxlan.ToMechanism(request.GetMechanismPreferences()[0])
	assert.NotNil(t, mechanism)
	assert.Equal(t, "192.0.2.1", mechanism.SrcIP().String())
	assert.Equal(t, port, mechanism.SrcPort())
}
