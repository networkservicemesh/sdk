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

package grpcutils

import (
	"fmt"
	"net"
	"net/url"
)

// URLToTarget - convert *net.URL to acceptable grpc target value.
func URLToTarget(u *url.URL) (target string) {
	switch u.Scheme {
	case unixScheme:
		return u.String()
	case tcpScheme:
		return u.Path
	}
	// assume other variants converters just fine.
	return u.String()
}

// AddressToURL - convert net.Addr to a proper URL object
func AddressToURL(addr net.Addr) *url.URL {
	if tcpAddr, ok := addr.(*net.TCPAddr); ok {
		if tcpAddr.IP.IsUnspecified() {
			return &url.URL{Scheme: addr.Network(), Path: fmt.Sprintf("127.0.0.1:%v", tcpAddr.Port)}
		}
	}
	return &url.URL{Scheme: addr.Network(), Path: addr.String()}
}
