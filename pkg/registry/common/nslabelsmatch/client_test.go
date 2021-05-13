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

package nslabelsmatch_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/common/memory"
	"github.com/networkservicemesh/sdk/pkg/registry/common/nslabelsmatch"
	"github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/registry/core/chain"
)

const (
	testEnvName                      = "NODE_NAME"
	testEnvValue                     = "testValue"
	destinationTestKey               = "nodeName"
	destinationTestCorrectTemplate   = `{{index .Src "NodeNameKey"}}`
	destinationTestIncorrectTemplate = `{{index .Src "SomeRandomKey"}}`
	destinationTestNotATemplate      = "SomeRandomString"
)

func TestNSTemplates(t *testing.T) {
	err := os.Setenv(testEnvName, testEnvValue)
	require.NoError(t, err)

	tests := []struct {
		name     string
		template string
		want     string
	}{
		{
			name:     "CorrectTemplate",
			template: destinationTestCorrectTemplate,
			want:     testEnvValue,
		},
		{
			name:     "IncorrectTemplate",
			template: destinationTestIncorrectTemplate,
			want:     "",
		},
		{
			name:     "NotATemplate",
			template: destinationTestNotATemplate,
			want:     destinationTestNotATemplate,
		},
	}

	for _, tc := range tests {
		// nolint:scopelint
		t.Run(tc.name, func(t *testing.T) {
			testNSTemplate(t, tc.template, tc.want)
		})
	}
}

func testNSTemplate(t *testing.T, template, want string) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	nsrc := chain.NewNetworkServiceRegistryClient(
		nslabelsmatch.NewNetworkServiceRegistryClient(ctx),
		adapters.NetworkServiceServerToClient(memory.NewNetworkServiceRegistryServer()),
	)

	nsReg := &registry.NetworkService{
		Matches: []*registry.Match{
			{
				Routes: []*registry.Destination{
					{
						DestinationSelector: map[string]string{
							destinationTestKey: template,
						},
					},
				},
			},
		},
	}

	nsReg, err := nsrc.Register(ctx, nsReg)
	require.NoError(t, err)

	stream, err := nsrc.Find(ctx, &registry.NetworkServiceQuery{NetworkService: nsReg})
	require.NoError(t, err)

	nsReg, err = stream.Recv()
	require.NoError(t, err)

	require.Equal(t, want, nsReg.Matches[0].Routes[0].DestinationSelector[destinationTestKey])
}
