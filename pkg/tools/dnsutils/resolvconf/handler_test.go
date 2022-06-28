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

package resolvconf

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"golang.org/x/net/context"
)

func TestResolvConf(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	_, cancel := context.WithTimeout(context.Background(), time.Microsecond)
	defer cancel()

	file, err := os.CreateTemp(os.TempDir(), "resolv.conf")
	require.NoError(t, err)
	configPath := file.Name()
	err = file.Close()
	require.NoError(t, err)

	conf, err := openResolveConfig(configPath)
	require.NoError(t, err)

	conf.SetValue("nameserver", "8.8.8.8")

	err = conf.Save()
	require.NoError(t, err)

	NewDNSHandler(WithResolveConfigPath(configPath))

	conf, err = openResolveConfig(configPath)
	require.NoError(t, err)

	value := conf.Value("nameserver")
	require.Equal(t, value, []string{"127.0.0.1"})
}
