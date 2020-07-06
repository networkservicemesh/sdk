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

package cidr

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNetworkAddress(t *testing.T) {
	_, ipnet, _ := net.ParseCIDR("192.168.3.4/16")
	assert.Equal(t, "192.168.0.0", NetworkAddress(ipnet).String())
}

func TestBroadcastAddress(t *testing.T) {
	_, ipnet, _ := net.ParseCIDR("192.168.3.4/16")
	assert.Equal(t, "192.168.255.255", BroadcastAddress(ipnet).String())
}

func Test24(t *testing.T) {
	_, ipnet, _ := net.ParseCIDR("192.168.1.0/24")
	assert.Equal(t, "192.168.1.0", NetworkAddress(ipnet).String())
	assert.Equal(t, "192.168.1.255", BroadcastAddress(ipnet).String())
}

func Test32(t *testing.T) {
	_, ipnet, _ := net.ParseCIDR("192.168.1.1/32")
	assert.Equal(t, "192.168.1.1", NetworkAddress(ipnet).String())
	assert.Equal(t, "192.168.1.1", BroadcastAddress(ipnet).String())
}
