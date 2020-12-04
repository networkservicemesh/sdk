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

package metadatahelper_test

import (
	"context"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/updatepath"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/checks/checkcontext"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	"github.com/networkservicemesh/sdk/pkg/tools/metadatahelper"
)

type sample struct {
	name string
	test func(t *testing.T, ctx context.Context)
}

var samples = []*sample{
	{
		name: "Store + Load",
		test: func(t *testing.T, ctx context.Context) {
			metaData := metadatahelper.IntMetadata(ctx, false)

			expected := 10
			metaData.Store(expected)

			actual, ok := metaData.Load()
			require.True(t, ok)
			require.Equal(t, expected, actual)
		},
	},
	{
		name: "Store + Delete",
		test: func(t *testing.T, ctx context.Context) {
			metaData := metadatahelper.IntMetadata(ctx, false)

			metaData.Store(10)
			metaData.Delete()

			_, ok := metaData.Load()
			require.False(t, ok)
		},
	},
}

func TestMetadata(t *testing.T) {
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
