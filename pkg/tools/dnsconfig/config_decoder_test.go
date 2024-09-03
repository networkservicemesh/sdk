// Copyright (c) 2021 Doc.ai and/or its affiliates.
//
// Copyright (c) 2022 Cisco and/or its affiliates.
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

package dnsconfig_test

import (
	"testing"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/tools/dnsconfig"
)

func Test_DNSConfigsDecoder_ShouldBeParsedFromJson(t *testing.T) {
	expected := []*networkservice.DNSConfig{
		{
			SearchDomains: []string{"d1"},
			DnsServerIps:  []string{"ip1", "ip2"},
		},
	}
	var decoder dnsconfig.Decoder
	err := decoder.Decode(`[{"dns_server_ips": ["ip1", "ip2"], "search_domains": ["d1"]}]`)
	require.NoError(t, err)
	require.Equal(t, expected, []*networkservice.DNSConfig(decoder))
}
