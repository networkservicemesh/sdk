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

package maphelper_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/updatepath"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/checks/checkcontext"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

const (
	key1   = "a"
	key2   = "b"
	value1 = 10
	value2 = 20
)

type sample struct {
	name string
	test func(t *testing.T, ctx context.Context)
}

var samples = []*sample{
	{
		name: "Store + Load",
		test: func(t *testing.T, ctx context.Context) {
			mapMetaData := testMapMetaData(ctx, false)

			mapMetaData.Store(key1, value1)
			mapMetaData.Store(key2, value2)

			actual, ok := mapMetaData.Load(key1)
			require.True(t, ok)
			require.Equal(t, value1, actual)

			actual, ok = mapMetaData.Load(key2)
			require.True(t, ok)
			require.Equal(t, value2, actual)
		},
	},
	{
		name: "Store + Delete",
		test: func(t *testing.T, ctx context.Context) {
			mapMetaData := testMapMetaData(ctx, false)

			mapMetaData.Store(key1, value1)
			mapMetaData.Delete(key1)

			_, ok := mapMetaData.Load(key1)
			require.False(t, ok)
		},
	},
	{
		name: "Store + raw Load",
		test: func(t *testing.T, ctx context.Context) {
			mapMetaData := testMapMetaData(ctx, false)
			rawMetaData := metadata.Map(ctx, false)

			mapMetaData.Store(key1, value1)

			_, ok := rawMetaData.Load(key1)
			require.False(t, ok)
		},
	},
}

func TestMapMetaData(t *testing.T) {
	for i := range samples {
		sample := samples[i]
		t.Run(sample.name, func(t *testing.T) {
			var requested bool
			server := next.NewNetworkServiceServer(
				updatepath.NewServer("server"),
				metadata.NewServer(),
				checkcontext.NewServer(t, func(t *testing.T, ctx context.Context) {
					sample.test(t, ctx)
					requested = true
				}),
			)
			_, _ = server.Request(context.TODO(), new(networkservice.NetworkServiceRequest))
			require.True(t, requested)
		})
	}
}
