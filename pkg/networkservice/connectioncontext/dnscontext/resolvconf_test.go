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

package dnscontext

import (
	"os"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func createSample(name, source string) (string, error) {
	tmpPath := path.Join(os.TempDir(), name)
	err := os.WriteFile(tmpPath, []byte(source), os.ModePerm)
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
	conf, err := openResolveConfig(p)
	require.NotNil(t, conf)
	require.Nil(t, err)
	result := conf.Value(nameserverProperty)
	require.EqualValues(t, result, []string{"127.0.0.1"})
	result = conf.Value(searchProperty)
	require.EqualValues(t, result, []string{"default.svc.cluster.local", "svc.cluster.local", "cluster.local"})
	result = conf.Value(optionsProperty)
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
	config, err := openResolveConfig(p)
	require.Nil(t, err)
	config.SetValue(nameserverProperty, "127.0.0.1")
	config.SetValue(searchProperty, "default.svc.cluster.local", "svc.cluster.local", "cluster.local")
	config.SetValue(optionsProperty, "ndots:5")
	config.SetValue("my_property")
	actual := config.String()
	require.Len(t, actual, len(sample))
	for _, l := range strings.Split(actual, "\n") {
		require.Contains(t, sample, l)
	}
}
