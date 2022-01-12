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

package dnscontext_test

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/tools/dnscontext"
)

const (
	optionsKey = "options"
	searchKey  = "search"
	sample     = `nameserver 1.1.1.1
nameserver 2.2.2.2
options ndots:5
search default.svc.cluster.local svc.cluster.local cluster.local
`
)

func createSample(t *testing.T, source string) string {
	tmpPath := path.Join(t.TempDir(), "resolv.conf")
	require.NoError(t, ioutil.WriteFile(tmpPath, []byte(source), os.ModePerm))
	return tmpPath
}

func TestResolveConfig_ReadProperties(t *testing.T) {
	p := createSample(t, sample)
	t.Cleanup(func() { _ = os.Remove(p) })

	conf, err := dnscontext.OpenResolveConfig(p)
	require.NoError(t, err)
	require.NotNil(t, conf)

	require.Equal(t, []string{"1.1.1.1", "2.2.2.2"}, conf.NameServers)
	require.Equal(t, []string{"ndots:5"}, conf.Value(optionsKey))
	require.Equal(t, []string{"default.svc.cluster.local", "svc.cluster.local", "cluster.local"}, conf.Value(searchKey))
}

func TestResolveConfig_WriteProperties(t *testing.T) {
	p := createSample(t, "")
	t.Cleanup(func() { _ = os.Remove(p) })

	conf, err := dnscontext.OpenResolveConfig(p)
	require.NoError(t, err)
	require.NotNil(t, conf)

	conf.NameServers = []string{"1.1.1.1", "2.2.2.2"}
	conf.SetValue(optionsKey, "ndots:5")
	conf.SetValue(searchKey, "default.svc.cluster.local", "svc.cluster.local", "cluster.local")
	conf.SetValue("empty")
	require.NoError(t, conf.Save())

	bytes, err := ioutil.ReadFile(filepath.Clean(p))
	require.NoError(t, err)
	require.Equal(t, sample, string(bytes))
}
