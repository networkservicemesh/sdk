// Copyright (c) 2021 Doc.ai and/or its affiliates.
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

package label_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/label"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/null"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/switchcase"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/updatepath"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/checks/checkcontext"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

func TestLabelClient(t *testing.T) {
	const labelKey = "key"
	const labelValue = "value"

	c := next.NewNetworkServiceClient(
		updatepath.NewClient("name"),
		metadata.NewClient(),
		label.NewClient(labelKey, labelValue),
		switchcase.NewClient(
			&switchcase.ClientCase{
				Condition: label.Condition(labelKey, "invalid"),
				Client: checkcontext.NewClient(t, func(t *testing.T, _ context.Context) {
					require.Fail(t, "invalid label selected for client branch")
				}),
			},
			&switchcase.ClientCase{
				Condition: label.Condition(labelKey, labelValue),
				Client:    null.NewClient(),
			},
			&switchcase.ClientCase{
				Condition: switchcase.Default,
				Client: checkcontext.NewClient(t, func(t *testing.T, _ context.Context) {
					require.Fail(t, "no label selected for client branch")
				}),
			},
		),
		adapters.NewServerToClient(
			switchcase.NewServer(
				&switchcase.ServerCase{
					Condition: label.Condition(labelKey, "invalid"),
					Server: checkcontext.NewServer(t, func(t *testing.T, _ context.Context) {
						require.Fail(t, "invalid label selected for server branch")
					}),
				},
				&switchcase.ServerCase{
					Condition: label.Condition(labelKey, labelValue),
					Server:    null.NewServer(),
				},
				&switchcase.ServerCase{
					Condition: switchcase.Default,
					Server: checkcontext.NewServer(t, func(t *testing.T, _ context.Context) {
						require.Fail(t, "no label selected for server branch")
					}),
				},
			),
		),
	)

	_, err := c.Request(context.Background(), new(networkservice.NetworkServiceRequest))
	require.NoError(t, err)

	_, err = c.Close(context.Background(), new(networkservice.Connection))
	require.NoError(t, err)
}
