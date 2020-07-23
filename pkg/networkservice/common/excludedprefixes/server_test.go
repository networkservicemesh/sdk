// Copyright (c) 2020 Doc.ai and/or its affiliates.
//
// Copyright (c) 2020 Cisco and/or its affiliates.
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
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/checks/checkrequest"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

func TestNewExcludedPrefixesService(t *testing.T) {
	prefixes := []string{"172.16.1.0/24", "10.32.0.0/12", "10.96.0.0/12"}

	testConfig := strings.Join(append([]string{"prefixes:"}, prefixes...), "\n- ")
	configPath := path.Join(os.TempDir(), "excluded_prefixes.yaml")
	writingToFile(t, testConfig, configPath)
	defer func() { _ = os.Remove(configPath) }()

	chain := next.NewNetworkServiceServer(NewServer(WithConfigPath(configPath)), checkrequest.NewServer(t, func(t *testing.T, request *networkservice.NetworkServiceRequest) {
		require.Equal(t, request.Connection.Context.IpContext.ExcludedPrefixes, prefixes)
	}))
	req := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Context: &networkservice.ConnectionContext{},
		},
	}

	_, err := chain.Request(context.Background(), req)
	require.NoError(t, err)
}

func TestCheckReloadedPrefixes(t *testing.T) {
	prefixes := []string{"172.16.1.0/24", "10.32.0.0/12", "10.96.0.0/12"}

	testConfig := strings.Join(append([]string{"prefixes:"}, prefixes...), "\n- ")
	configPath := path.Join(os.TempDir(), "excluded_prefixes.yaml")
	writingToFile(t, "", configPath)
	defer func() { _ = os.Remove(configPath) }()

	chain := next.NewNetworkServiceServer(NewServer(WithConfigPath(configPath)))
	req := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Context: &networkservice.ConnectionContext{},
		},
	}

	err := ioutil.WriteFile(configPath, []byte(testConfig), 0600)
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		_, err = chain.Request(context.Background(), req)
		require.NoError(t, err)
		return reflect.DeepEqual(req.GetConnection().GetContext().GetIpContext().GetExcludedPrefixes(), prefixes)
	}, time.Second, time.Millisecond*100)
}

func TestUniqueRequestPrefixes(t *testing.T) {
	prefixes := []string{"172.16.1.0/24", "10.32.0.0/12", "10.96.0.0/12", "10.20.128.0/17", "10.20.64.0/18", "10.20.8.0/21", "10.20.4.0/22"}
	reqPrefixes := []string{"100.1.1.0/13", "10.32.0.0/12", "10.96.0.0/12", "10.20.0.0/24", "10.20.128.0/17", "10.20.64.0/18", "10.20.16.0/20", "10.20.2.0/23"}
	uniquePrefixes := []string{"100.1.1.0/13", "10.32.0.0/12", "10.96.0.0/12", "10.20.0.0/24", "10.20.128.0/17", "10.20.64.0/18", "10.20.16.0/20", "10.20.2.0/23", "172.16.1.0/24", "10.20.8.0/21", "10.20.4.0/22"}

	testConfig := strings.Join(append([]string{"prefixes:"}, prefixes...), "\n- ")
	configPath := path.Join(os.TempDir(), "excluded_prefixes.yaml")
	writingToFile(t, testConfig, configPath)
	defer func() { _ = os.Remove(configPath) }()

	chain := next.NewNetworkServiceServer(NewServer(WithConfigPath(configPath)), checkrequest.NewServer(t, func(t *testing.T, request *networkservice.NetworkServiceRequest) {
		require.Equal(t, uniquePrefixes, request.Connection.Context.IpContext.ExcludedPrefixes)
	}))
	req := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Context: &networkservice.ConnectionContext{
				IpContext: &networkservice.IPContext{
					ExcludedPrefixes: reqPrefixes,
				},
			},
		},
	}

	_, err := chain.Request(context.Background(), req)
	require.NoError(t, err)
}

func writingToFile(t *testing.T, text, configPath string) {
	f, err := os.Create(configPath)
	require.NoError(t, err)
	_, err = f.WriteString(text)
	require.NoError(t, err)
	err = f.Close()
	require.NoError(t, err)
}
