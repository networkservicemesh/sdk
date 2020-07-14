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

package capturecontext_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/capturecontext"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

func TestServerContextStorage(t *testing.T) {
	chain := adapters.NewClientToServer(next.NewNetworkServiceClient(&writeClient{}, capturecontext.NewClient(), &checkContextClient{}))

	ctx := context.WithValue(context.Background(), writerKey, true)
	_, err := chain.Request(ctx, nil)
	require.NoError(t, err)

	_, err = chain.Close(ctx, nil)
	require.NoError(t, err)
}
