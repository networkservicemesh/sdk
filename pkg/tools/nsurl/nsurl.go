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

// Package nsurl provides a wrapper for a url.URL that is being used to represent a Network Service being requested
// by a workload.
// The rough schema is
// ${mechanism}://${network service name}[/${interface name}][{labels in URL query format}]
// Examples:
//   kernel://my-service@dc.example.com/ms1
//   memif://my-vpp-service?A=1&B=2&C=3
//   vfio://second-service?sriovToken=intel/10G
package nsurl

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/common"
)

// NSURL - wrapper around *url.URL to allow easy extraction of NSM related information
type NSURL url.URL

// Mechanism - return Mechanism for the requested Network Service
func (n *NSURL) Mechanism() *networkservice.Mechanism {
	mechanism := &networkservice.Mechanism{Cls: cls.LOCAL, Type: strings.ToUpper(n.Scheme)}
	segments := strings.Split(n.Path, "/")

	if len(segments) > 1 {
		mechanism.Parameters = make(map[string]string)
		mechanism.Parameters[common.InterfaceNameKey] = segments[len(segments)-1]
	}
	return mechanism
}

// Labels - return the source Labels for the requested Network Service
func (n *NSURL) Labels() map[string]string {
	labels := make(map[string]string)
	for k, values := range (*url.URL)(n).Query() {
		labels[k] = strings.Join(values, ",")
	}
	return labels
}

// NetworkService - return the name of the requested Network Service
func (n *NSURL) NetworkService() string {
	if n.User.Username() != "" {
		return fmt.Sprintf("%s@%s", n.User.Username(), (*url.URL)(n).Hostname())
	}
	return (*url.URL)(n).Hostname()
}
