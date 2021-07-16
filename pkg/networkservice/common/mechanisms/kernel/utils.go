// Copyright (c) 2021 Doc.ai and/or its affiliates.
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

package kernel

import (
	"fmt"
	"net/url"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	kernelmech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
)

var netNSURL = (&url.URL{Scheme: "file", Path: "/proc/thread-self/ns/net"}).String()

// getNameFromConnection - returns a name computed from networkservice.Connection 'conn'
func getNameFromConnection(conn *networkservice.Connection) string {
	ns := conn.GetNetworkService()
	nsMaxLength := kernelmech.LinuxIfMaxLength - 5
	if len(ns) > nsMaxLength {
		ns = ns[:nsMaxLength]
	}
	name := fmt.Sprintf("%s-%s", ns, conn.GetId())
	return limitName(name)
}

func limitName(name string) string {
	if len(name) > kernelmech.LinuxIfMaxLength {
		return name[:kernelmech.LinuxIfMaxLength]
	}
	return name
}
