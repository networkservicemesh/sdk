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

package filtermechanisms_test

import (
	"context"
	"net/url"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/memif"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/srv6"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vxlan"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/filtermechanisms"
	"github.com/networkservicemesh/sdk/pkg/tools/clienturlctx"
)

func TestFilterMechanismsClient_Request(t *testing.T) {
	request := func() *networkservice.NetworkServiceRequest {
		return &networkservice.NetworkServiceRequest{
			MechanismPreferences: []*networkservice.Mechanism{
				{
					Cls:  cls.REMOTE,
					Type: srv6.MECHANISM,
				},
				{
					Cls:  cls.REMOTE,
					Type: vxlan.MECHANISM,
				},
				{
					Cls:  cls.LOCAL,
					Type: kernel.MECHANISM,
				},
				{
					Cls:  cls.LOCAL,
					Type: memif.MECHANISM,
				},
			},
		}
	}
	samples := []struct {
		Name         string
		EndpointName string
		ClientURL    *url.URL
		RegisterURLs []url.URL
		ClsResult    string
	}{
		{
			Name:         "Local mechanisms",
			ClientURL:    &url.URL{Scheme: "inode", Host: "localhost:5000"},
			EndpointName: "nse-1",
			ClsResult:    cls.LOCAL,
		},
		{
			Name:         "Remote mechanisms",
			ClientURL:    &url.URL{Scheme: "tcp", Host: "localhost:5000"},
			EndpointName: "nse-1",
			ClsResult:    cls.REMOTE,
		},
	}

	for _, sample := range samples {
		s := filtermechanisms.NewClient()
		ctx := clienturlctx.WithClientURL(context.Background(), sample.ClientURL)
		req := request()
		_, err := s.Request(ctx, req)
		require.NoError(t, err)
		require.NotEmpty(t, req.GetMechanismPreferences())

		if sample.ClsResult != "" {
			for _, m := range req.GetMechanismPreferences() {
				require.Equal(t, sample.ClsResult, m.GetCls(), "filtermechanisms chain element should properly filter mechanisms")
			}
		} else {
			require.Equal(t, request().String(), req.String())
		}
	}
}
