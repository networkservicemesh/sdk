// Copyright (c) 2021 Doc.ai and/or its affiliates.
//
// Copyright (c) 2024 Cisco and/or its affiliates.
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

	kernelmech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"

	"github.com/networkservicemesh/sdk/pkg/tools/nanoid"
)

const (
	ifPrefix = "nsm"
)

var netNSURL = (&url.URL{Scheme: "file", Path: "/proc/thread-self/ns/net"}).String()

// generateInterfaceName - returns a random interface name with "nsm" prefix
func generateInterfaceName() (string, error) {
	ifIDLen := kernelmech.LinuxIfMaxLength - len(ifPrefix)
	id, err := nanoid.RandomString(ifIDLen)
	if err != nil {
		return "", err
	}
	name := fmt.Sprintf("%s%s", ifPrefix, id)

	return limitName(name), nil
}

func limitName(name string) string {
	if len(name) > kernelmech.LinuxIfMaxLength {
		return name[:kernelmech.LinuxIfMaxLength]
	}
	return name
}
