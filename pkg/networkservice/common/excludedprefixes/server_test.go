// Copyright (c) 2020 Cisco Systems, Inc.
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

package excludedprefixes

import (
	"context"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/checks/checkrequest"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

func TestNewExcludedPrefixesService(t *testing.T) {
	prefixes := []string{"172.16.1.0/24", "10.32.0.0/12", "10.96.0.0/12"}
	testConfig := strings.Join(append([]string{"prefixes:"}, prefixes...), "\n- ")

	configPath := path.Join(os.TempDir(), "excluded_prefixes.yaml")
	f, err := os.Create(configPath)
	require.Nil(t, err)
	defer func() { _ = os.Remove(configPath) }()
	_, err = f.WriteString(testConfig)
	require.Nil(t, err)
	err = f.Close()
	require.Nil(t, err)

	chain := next.NewNetworkServiceServer(NewServerFromPath(configPath), checkrequest.NewServer(t, func(t *testing.T, request *networkservice.NetworkServiceRequest) {
		require.Equal(t, request.Connection.Context.IpContext.ExcludedPrefixes, prefixes)
	}))
	req := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Context: &networkservice.ConnectionContext{
				IpContext: &networkservice.IPContext{},
			},
		},
	}
	_, _ = chain.Request(context.Background(), req)
}
