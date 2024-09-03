// Copyright (c) 2020-2022 Cisco and/or its affiliates.
//
// Copyright (c) 2020-2022 Doc.ai and/or its affiliates.
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

package excludedprefixes_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/excludedprefixes"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/checks/checkrequest"
)

const defaultPrefixesFileName = "excluded_prefixes.yaml"

func TestNewExcludedPrefixesService(t *testing.T) {
	prefixes := []string{"10.32.0.0/12", "10.96.0.0/12", "172.16.1.0/24"}

	dir := filepath.Join(os.TempDir(), t.Name())
	defer func() { _ = os.RemoveAll(dir) }()
	require.NoError(t, os.MkdirAll(dir, os.ModePerm))
	testConfig := strings.Join(append([]string{"prefixes:"}, prefixes...), "\n- ")
	configPath := filepath.Join(dir, defaultPrefixesFileName)
	require.NoError(t, os.WriteFile(configPath, []byte(testConfig), os.ModePerm))

	server := chain.NewNetworkServiceServer(excludedprefixes.NewServer(context.Background(), excludedprefixes.WithConfigPath(configPath)), checkrequest.NewServer(t, func(t *testing.T, request *networkservice.NetworkServiceRequest) {
		require.Equal(t, request.GetConnection().GetContext().GetIpContext().GetExcludedPrefixes(), prefixes)
	}))
	req := request()

	_, err := server.Request(context.Background(), req)
	require.NoError(t, err)
}

func TestCheckReloadedPrefixes(t *testing.T) {
	prefixes := []string{"10.32.0.0/12", "10.96.0.0/12", "172.16.1.0/24"}

	dir := filepath.Join(os.TempDir(), t.Name())
	defer func() { _ = os.RemoveAll(dir) }()
	require.NoError(t, os.MkdirAll(dir, os.ModePerm))

	testConfig := strings.Join(append([]string{"prefixes:"}, prefixes...), "\n- ")
	configPath := filepath.Join(dir, defaultPrefixesFileName)
	require.NoError(t, os.WriteFile(configPath, []byte(""), os.ModePerm))

	defer func() { _ = os.Remove(configPath) }()

	server := chain.NewNetworkServiceServer(excludedprefixes.NewServer(context.Background(), excludedprefixes.WithConfigPath(configPath)))
	req := request()

	err := os.WriteFile(configPath, []byte(testConfig), 0o600)
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		_, err = server.Request(context.Background(), req)
		require.NoError(t, err)
		return reflect.DeepEqual(req.GetConnection().GetContext().GetIpContext().GetExcludedPrefixes(), prefixes)
	}, time.Second*15, time.Millisecond*100)
}

func TestExcludedPrefixesServer(t *testing.T) {
	t.Run("Handle not created yet prefixes file", func(t *testing.T) {
		dir := filepath.Join(os.TempDir(), t.Name())
		defer func() { _ = os.RemoveAll(dir) }()
		require.NoError(t, os.MkdirAll(dir, os.ModePerm))
		filePath := filepath.Join(dir, defaultPrefixesFileName)
		_ = os.Remove(filePath)
		testWaitForFile(t, filePath)
	})

	t.Run("Handle prefixes file with empty directory path", func(t *testing.T) {
		_ = os.Remove(defaultPrefixesFileName)
		testWaitForFile(t, defaultPrefixesFileName)
	})
}

func TestUniqueRequestPrefixes(t *testing.T) {
	prefixes := []string{"172.16.1.0/24", "10.32.0.0/12", "10.96.0.0/12", "10.20.128.0/17", "10.20.64.0/18", "10.20.8.0/21", "10.20.4.0/22"}
	reqPrefixes := []string{"100.1.1.0/13", "10.32.0.0/12", "10.96.0.0/12", "10.20.0.0/24", "10.20.128.0/17", "10.20.64.0/18", "10.20.16.0/20", "10.20.2.0/23"}
	uniquePrefixes := []string{"100.1.1.0/13", "10.32.0.0/12", "10.96.0.0/12", "10.20.0.0/24", "10.20.128.0/17", "10.20.64.0/18", "10.20.16.0/20", "10.20.2.0/23", "10.20.4.0/22", "10.20.8.0/21", "172.16.1.0/24"}

	dir := filepath.Join(os.TempDir(), t.Name())
	defer func() { _ = os.RemoveAll(dir) }()
	require.NoError(t, os.MkdirAll(dir, os.ModePerm))

	testConfig := strings.Join(append([]string{"prefixes:"}, prefixes...), "\n- ")
	configPath := filepath.Join(dir, defaultPrefixesFileName)
	require.NoError(t, os.WriteFile(configPath, []byte(testConfig), os.ModePerm))
	defer func() { _ = os.Remove(configPath) }()

	server := chain.NewNetworkServiceServer(excludedprefixes.NewServer(context.Background(), excludedprefixes.WithConfigPath(configPath)), checkrequest.NewServer(t, func(t *testing.T, request *networkservice.NetworkServiceRequest) {
		require.Equal(t, uniquePrefixes, request.GetConnection().GetContext().GetIpContext().GetExcludedPrefixes())
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

	_, err := server.Request(context.Background(), req)
	require.NoError(t, err)
}

func TestFilePrefixesChanged(t *testing.T) {
	filePrefixes := []string{"172.16.1.0/24", "10.32.0.0/12", "10.96.0.0/12", "10.20.128.0/17"}
	reqPrefixes := []string{"100.1.1.0/13", "10.32.0.0/12", "10.96.0.0/12"}
	diffPrefxies := []string{"100.1.1.0/13", "10.32.0.0/12", "10.96.0.0/12", "10.20.128.0/17", "172.16.1.0/24"}
	newFilePrefixes := []string{"172.16.2.0/24"}

	dir := filepath.Join(os.TempDir(), t.Name())
	defer func() { _ = os.RemoveAll(dir) }()
	require.NoError(t, os.MkdirAll(dir, os.ModePerm))

	testConfig := strings.Join(append([]string{"prefixes:"}, filePrefixes...), "\n- ")
	configPath := filepath.Join(dir, defaultPrefixesFileName)
	require.NoError(t, os.WriteFile(configPath, []byte(testConfig), os.ModePerm))
	defer func() { _ = os.Remove(configPath) }()

	var prefixesAfterServer []string
	isFirst := true
	server := chain.NewNetworkServiceServer(
		excludedprefixes.NewServer(context.Background(), excludedprefixes.WithConfigPath(configPath)),
		checkrequest.NewServer(t, func(t *testing.T, request *networkservice.NetworkServiceRequest) {
			if isFirst {
				require.ElementsMatch(t, diffPrefxies, request.GetConnection().GetContext().GetIpContext().GetExcludedPrefixes())
				isFirst = false
			}

			prefixesAfterServer = request.GetConnection().GetContext().GetIpContext().GetExcludedPrefixes()
		}),
	)

	req := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: "1",
			Context: &networkservice.ConnectionContext{
				IpContext: &networkservice.IPContext{
					ExcludedPrefixes: reqPrefixes,
				},
			},
		},
	}

	_, err := server.Request(context.Background(), req)

	require.NoError(t, os.WriteFile(configPath, []byte(strings.Join(append([]string{"prefixes:"}, newFilePrefixes...), "\n- ")), os.ModePerm))

	require.Eventually(t, func() bool {
		_, err = server.Request(context.Background(), req)
		fmt.Printf("%v\n", prefixesAfterServer)
		return cmp.Equal(prefixesAfterServer, append(diffPrefxies, newFilePrefixes...), cmp.Transformer("Sort", func(in []string) []string {
			out := append([]string(nil), in...)
			sort.Strings(out)
			return out
		}))
	}, time.Second, time.Millisecond*100)

	require.NoError(t, err)
}

func testWaitForFile(t *testing.T, filePath string) {
	prefixes := []string{"10.80.0.0/12", "172.16.1.0/24"}

	testConfig := strings.Join(append([]string{"prefixes:"}, prefixes...), "\n- ")

	server := chain.NewNetworkServiceServer(excludedprefixes.NewServer(context.Background(),
		excludedprefixes.WithConfigPath(filePath)))

	req := request()
	_, err := server.Request(context.Background(), req)
	require.NoError(t, err)
	require.Empty(t, req.GetConnection().GetContext().GetIpContext().GetExcludedPrefixes())

	require.NoError(t, os.WriteFile(filePath, []byte(testConfig), os.ModePerm))

	defer func() { _ = os.Remove(filePath) }()
	require.Eventually(t, func() bool {
		_, reqErr := server.Request(context.Background(), req)
		require.NoError(t, reqErr)
		return reflect.DeepEqual(req.GetConnection().GetContext().GetIpContext().GetExcludedPrefixes(), prefixes)
	}, time.Second, time.Millisecond*100)

	err = os.Remove(filePath)
	require.Nil(t, err)

	require.Eventually(t, func() bool {
		req := request()
		_, reqErr := server.Request(context.Background(), req)
		require.NoError(t, reqErr)
		return len(req.GetConnection().GetContext().GetIpContext().GetExcludedPrefixes()) == 0
	}, time.Second, time.Millisecond*100)
}

func request() *networkservice.NetworkServiceRequest {
	return &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Context: &networkservice.ConnectionContext{},
		},
	}
}
