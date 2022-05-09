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

// Package listenonurl providesfunctions to set public url to register NSM
package listenonurl

import (
	"net"
	"net/url"
)

// GetPublicURL - constructs an URL from an address accessible
//  clusterwide and a given port
// 	defaults to the given URL if no such address can be found
func GetPublicURL(addrs []net.Addr, defaultURL *url.URL) *url.URL {
	if defaultURL.Port() == "" || len(defaultURL.Host) != len(":")+len(defaultURL.Port()) {
		return defaultURL
	}

	if listenIP := getIP(addrs); listenIP != nil {
		return &url.URL{
			Scheme: "tcp",
			Host:   net.JoinHostPort(listenIP.String(), defaultURL.Port()),
		}
	}
	return defaultURL
}

func getIP(addrs []net.Addr) net.IP {
	var ret net.IP
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			// defaults to IPv4 address if available
			if ipnet.IP.To4() != nil {
				return ipnet.IP
			}
			// accept all IPv6 Global Unicats addresses
			// including Unique-Local Address (which starts with ‘fd’)
			if ret == nil && ipnet.IP.IsGlobalUnicast() {
				ret = ipnet.IP
			}
		}
	}
	return ret
}
