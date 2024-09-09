// Copyright (c) 2021-2022 Nordix and/or its affiliates.
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

package listenonurl_test

import (
	"net"
	"net/url"
	"testing"

	"github.com/networkservicemesh/sdk/pkg/tools/listenonurl"
)

func TestGetPublicURL(t *testing.T) {
	test := []struct {
		u   *url.URL
		ips []net.Addr
		exp string
	}{
		{
			u: &url.URL{
				Scheme: "file",
				Host:   "",
				Path:   "/local-path",
			},
			ips: []net.Addr{ // 10.244.2.13/24
				&net.IPNet{IP: net.IPv4(10, 244, 2, 13), Mask: net.IPv4Mask(255, 255, 255, 0)},
			},
			exp: "file:///local-path",
		}, {
			u: &url.URL{
				Scheme: "tcp",
				Host:   "172.17.11.1:5009",
			},
			ips: []net.Addr{ // 10.244.2.13/24
				&net.IPNet{IP: net.IPv4(10, 244, 2, 13), Mask: net.IPv4Mask(255, 255, 255, 0)},
			},
			exp: "tcp://172.17.11.1:5009",
		}, {
			u: &url.URL{
				Scheme: "tcp",
				Host:   ":5009",
			},
			ips: []net.Addr{ // 10.244.2.13/24
				&net.IPNet{IP: net.IPv4(10, 244, 2, 13), Mask: net.IPv4Mask(255, 255, 255, 0)},
			},
			exp: "tcp://10.244.2.13:5009",
		}, {
			u: &url.URL{
				Scheme: "tcp",
				Host:   ":5009",
			},
			ips: []net.Addr{ // 127.0.0.1/8
				&net.IPNet{IP: net.IPv4(127, 0, 0, 1), Mask: net.IPv4Mask(255, 0, 0, 0)},
			},
			exp: "tcp://:5009",
		}, {
			u: &url.URL{
				Scheme: "tcp",
				Host:   ":5009",
			},
			ips: []net.Addr{
				// 10.244.2.13/24, 127.0.0.1/8, 135.104.0.0/24
				&net.IPNet{IP: net.IPv4(10, 244, 2, 13), Mask: net.IPv4Mask(255, 255, 255, 0)},
				&net.IPNet{IP: net.IPv4(127, 0, 0, 1), Mask: net.IPv4Mask(255, 0, 0, 0)},
				&net.IPNet{IP: net.IPv4(135, 104, 0, 0), Mask: net.IPv4Mask(255, 255, 255, 0)},
			},
			exp: "tcp://10.244.2.13:5009",
		}, {
			u: &url.URL{
				Scheme: "tcp",
				Host:   ":5009",
			},
			ips: []net.Addr{
				// fd00:10:244:1::9/32, 10.244.2.13/24
				&net.IPNet{IP: net.ParseIP("fd00:10:244:1::9"), Mask: net.CIDRMask(32, 128)},
				&net.IPNet{IP: net.IPv4(10, 244, 2, 13), Mask: net.IPv4Mask(255, 255, 255, 0)},
			},
			exp: "tcp://10.244.2.13:5009",
		}, {
			u: &url.URL{
				Scheme: "tcp",
				Host:   ":5009",
			},
			ips: []net.Addr{ // fd00:10:244:1::9/32
				&net.IPNet{IP: net.ParseIP("fd00:10:244:1::9"), Mask: net.CIDRMask(32, 128)},
			},
			exp: "tcp://[fd00:10:244:1::9]:5009",
		}, {
			u: &url.URL{
				Scheme: "tcp",
				Host:   ":5009",
			},
			ips: []net.Addr{ // 2001:DB8::1/48, fd00:10:244:1::9/32
				&net.IPNet{IP: net.ParseIP("2001:DB8::1"), Mask: net.CIDRMask(48, 128)},
				&net.IPNet{IP: net.ParseIP("fd00:10:244:1::9"), Mask: net.CIDRMask(32, 128)},
			},
			exp: "tcp://[2001:db8::1]:5009",
		}, {
			u: &url.URL{
				Scheme: "tcp",
				Host:   ":5009",
			},
			ips: []net.Addr{ // fe80::1/48
				&net.IPNet{IP: net.ParseIP("fe80::1"), Mask: net.CIDRMask(48, 128)},
			},
			exp: "tcp://:5009",
		},
	}

	for _, tt := range test {
		actual := listenonurl.GetPublicURL(tt.ips, tt.u)
		if tt.exp != actual.String() {
			t.Errorf("GetPublicURL(%q, %q) = %q, want %q", tt.ips, tt.u, actual, tt.exp)
		}
	}
}
