// Copyright (c) 2020 Doc.ai and/or its affiliates.
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

package dnscontext_test

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/tools/dnscontext"
)

func createSample(name, source string) (string, error) {
	tmpPath := path.Join(os.TempDir(), name)
	err := ioutil.WriteFile(tmpPath, []byte(source), os.ModePerm)
	if err != nil {
		return "", err
	}
	return tmpPath, nil
}

func TestResolveConfig_ReadProperties(t *testing.T) {
	sample := `nameserver 127.0.0.1
search default.svc.cluster.local svc.cluster.local cluster.local
options ndots:5
`
	p, err := createSample("resolv.conf.tmp.test.file", sample)
	require.Nil(t, err)
	defer func() {
		_ = os.Remove(p)
	}()
	conf, err := dnscontext.OpenResolveConfig(p)
	require.NotNil(t, conf)
	require.Nil(t, err)
	result := conf.Value(dnscontext.NameserverProperty)
	require.EqualValues(t, result, []string{"127.0.0.1"})
	result = conf.Value(dnscontext.SearchProperty)
	require.EqualValues(t, result, []string{"default.svc.cluster.local", "svc.cluster.local", "cluster.local"})
	result = conf.Value(dnscontext.OptionsProperty)
	require.EqualValues(t, result, []string{"ndots:5"})
}

func TestResolveConfig_WriteProperties(t *testing.T) {
	sample := `nameserver 127.0.0.1
search default.svc.cluster.local svc.cluster.local cluster.local
options ndots:5`
	p, err := createSample("resolv.conf.tmp.test.file", "")
	require.Nil(t, err)
	defer func() {
		_ = os.Remove(p)
	}()
	config, err := dnscontext.OpenResolveConfig(p)
	require.Nil(t, err)
	config.SetValue(dnscontext.NameserverProperty, "127.0.0.1")
	config.SetValue(dnscontext.SearchProperty, "default.svc.cluster.local", "svc.cluster.local", "cluster.local")
	config.SetValue(dnscontext.OptionsProperty, "ndots:5")
	config.SetValue("my_property")
	err = config.Save()
	require.Nil(t, err)
	bytes, err := ioutil.ReadFile(filepath.Clean(p))
	require.Nil(t, err)
	actual := string(bytes)
	require.Len(t, actual, len(sample))
	for _, l := range strings.Split(actual, "\n") {
		require.Contains(t, sample, l)
	}
}
