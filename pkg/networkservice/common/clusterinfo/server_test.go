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

package clusterinfo_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clusterinfo"
)

func TestReadClusterName(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	path := filepath.Join(t.TempDir(), "clusterinfo.yaml")
	require.NoError(t, os.WriteFile(path, []byte("CLUSTER_NAME: my-cluster1"), os.ModePerm))

	s := clusterinfo.NewServer(clusterinfo.WithConfigPath(path))

	resp, err := s.Request(context.Background(), &networkservice.NetworkServiceRequest{Connection: &networkservice.Connection{}})
	require.NoError(t, err)

	require.Len(t, resp.GetLabels(), 1)
	require.Equal(t, "my-cluster1", resp.GetLabels()["CLUSTER_NAME"])
}
