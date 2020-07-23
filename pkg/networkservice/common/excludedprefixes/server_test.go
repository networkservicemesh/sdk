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

package excludedprefixes

import (
	"context"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"strings"
	"sync"
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
			Context: &networkservice.ConnectionContext{},
		},
	}

	_, _ = chain.Request(context.Background(), req)
}

func TestCheckReloadedPrefixes(t *testing.T) {
	var prefixesUpdated = &struct {
		sync.RWMutex
		b bool
	}{}
	prefixes := []string{"172.16.1.0/24", "10.32.0.0/12", "10.96.0.0/12"}
	testConfig := strings.Join(append([]string{"prefixes:"}, prefixes...), "\n- ")

	configPath := path.Join(os.TempDir(), "excluded_prefixes.yaml")
	f, err := os.Create(configPath)
	require.Nil(t, err)
	defer func() { _ = os.Remove(configPath) }()
	err = f.Close()
	require.Nil(t, err)

	chain := next.NewNetworkServiceServer(NewServerFromPath(configPath), checkrequest.NewServer(t, func(t *testing.T, request *networkservice.NetworkServiceRequest) {
		if reflect.DeepEqual(request.Connection.Context.IpContext.ExcludedPrefixes, prefixes) {
			prefixesUpdated.Lock()
			prefixesUpdated.b = true
			prefixesUpdated.Unlock()
		}
	}))
	req := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Context: &networkservice.ConnectionContext{},
		},
	}

	err = ioutil.WriteFile(configPath, []byte(testConfig), 0600)
	require.Nil(t, err)

	require.Eventually(t, func() bool {
		_, _ = chain.Request(context.Background(), req)
		prefixesUpdated.RLock()
		defer prefixesUpdated.RUnlock()
		v := prefixesUpdated.b
		return v
	}, time.Second, time.Millisecond*100)
}

func TestUniqueRequestPrefixes(t *testing.T) {
	prefixes := []string{"172.16.1.0/24", "10.32.0.0/12", "10.96.0.0/12", "10.20.128.0/17", "10.20.64.0/18", "10.20.8.0/21", "10.20.4.0/22"}
	testConfig := strings.Join(append([]string{"prefixes:"}, prefixes...), "\n- ")
	reqPrefixes := []string{"100.1.1.0/13", "10.32.0.0/12", "10.96.0.0/12", "10.20.0.0/24", "10.20.128.0/17", "10.20.64.0/18", "10.20.16.0/20", "10.20.2.0/23"}

	configPath := path.Join(os.TempDir(), "excluded_prefixes.yaml")
	f, err := os.Create(configPath)
	require.Nil(t, err)
	defer func() { _ = os.Remove(configPath) }()
	_, err = f.WriteString(testConfig)
	require.Nil(t, err)
	err = f.Close()
	require.Nil(t, err)

	chain := next.NewNetworkServiceServer(NewServerFromPath(configPath), checkrequest.NewServer(t, func(t *testing.T, request *networkservice.NetworkServiceRequest) {
		require.True(t, uniquePrefixes(request.Connection.Context.IpContext.ExcludedPrefixes))
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

	_, _ = chain.Request(context.Background(), req)
}

func uniquePrefixes(elements []string) bool {
	encountered := map[string]bool{}

	for index := range elements {
		if encountered[elements[index]] {
			return false
		}
		encountered[elements[index]] = true
	}
	return true
}
