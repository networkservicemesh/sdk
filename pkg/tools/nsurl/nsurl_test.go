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

package nsurl_test

import (
	"net/url"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/memif"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vfio"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/tools/nsurl"
)

func must(u *url.URL, err error) *url.URL {
	if err != nil {
		panic(err.Error())
	}
	return u
}
func Test_NSURL(t *testing.T) {
	var samples = []struct {
		u              *url.URL
		interfaceName  string
		networkService string
		mechanism      string
		labels         map[string]string
	}{
		{
			u:              must(url.Parse("vfio://my-service/nsm-1")),
			interfaceName:  "nsm-1",
			networkService: "my-service",
			mechanism:      vfio.MECHANISM,
		},
		{
			u:              must(url.Parse("vfio://second-service?sriovToken=intel/10G")),
			networkService: "second-service",
			mechanism:      vfio.MECHANISM,
			labels: map[string]string{
				"sriovToken": "intel/10G",
			},
		},
		{
			u:              must(url.Parse("memif://my-vpp-service?A=1&B=2&C=3")),
			networkService: "my-vpp-service",
			mechanism:      memif.MECHANISM,
			labels: map[string]string{
				"A": "1",
				"B": "2",
				"C": "3",
			},
		},
		{
			u:              must(url.Parse("kernel://my-service@dc.example.com/ms1")),
			networkService: "my-service@dc.example.com",
			mechanism:      kernel.MECHANISM,
			interfaceName:  "ms1",
		},
	}

	for _, sample := range samples {
		require.Equal(t, cls.LOCAL, (*nsurl.NSURL)(sample.u).Mechanism().Cls)
		require.Equal(t, sample.mechanism, (*nsurl.NSURL)(sample.u).Mechanism().Type)
		require.Equal(t, sample.networkService, (*nsurl.NSURL)(sample.u).NetworkService())
		require.Len(t, (*nsurl.NSURL)(sample.u).Labels(), len(sample.labels))
		if len(sample.labels) > 0 {
			require.Equal(t, sample.labels, (*nsurl.NSURL)(sample.u).Labels())
		}
	}
}
