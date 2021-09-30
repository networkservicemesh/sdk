// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
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

package grpcutils

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// URLToTarget - convert *net.URL to acceptable grpc target value.
func URLToTarget(u *url.URL) (target string) {
	if u == nil {
		return ""
	}
	switch u.Scheme {
	case unixScheme:
		return u.String()
	case tcpScheme:
		return u.Host
	}
	// assume other variants converters just fine.
	return u.String()
}

// AddressToURL - convert net.Addr to a proper URL object
func AddressToURL(addr net.Addr) *url.URL {
	if tcpAddr, ok := addr.(*net.TCPAddr); ok {
		if tcpAddr.IP.IsUnspecified() {
			return &url.URL{Scheme: addr.Network(), Host: fmt.Sprintf(":%v", tcpAddr.Port)}
		}
	}
	return NetworkAddressToURL(addr.Network(), addr.String())
}

// NetworkAddressToURL - convert a network + address to proper URL object
func NetworkAddressToURL(network, address string) *url.URL {
	if network == unixScheme {
		return &url.URL{Scheme: network, Path: address}
	}
	return &url.URL{Scheme: network, Host: address}
}

// TargetToURL - convert target to a proper URL object
func TargetToURL(address string) *url.URL {
	network, addr := TargetToNetAddr(address)
	return NetworkAddressToURL(network, addr)
}

// TargetToNetAddr returns the network and address from a GRPC target
func TargetToNetAddr(target string) (network, addr string) {
	// Borrowed with love from grpc.parseDialTarget https://github.com/grpc/grpc-go/blob/9aa97f9/rpc_util.go#L821
	network = "tcp"

	m1 := strings.Index(target, ":")
	m2 := strings.Index(target, ":/")

	// handle unix:addr which will fail with url.Parse
	if m1 >= 0 && m2 < 0 {
		if n := target[0:m1]; n == unixScheme {
			network = n
			addr = target[m1+1:]
			return network, addr
		}
	}
	if m2 >= 0 {
		t, err := url.Parse(target)
		if err != nil {
			return network, target
		}
		scheme := t.Scheme
		addr = t.Path
		if scheme == unixScheme {
			network = scheme
			if addr == "" {
				addr = t.Host
			}
			return network, addr
		}
	}

	return network, target
}
