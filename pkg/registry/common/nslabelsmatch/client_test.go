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
	destinationTestCorrectTemplate   = "{{index .Src \"NodeNameKey\"}}"
	destinationTestIncorrectTemplate = "{{index .Src \"SomeRandomKey\"}}"
	destinationTestNotATemplate      = "SomeRandomString"
)

func TestNSTemplates(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	err := os.Setenv(testEnvName, testEnvValue)
	require.NoError(t, err)

	mem := memory.NewNetworkServiceRegistryServer()
	nsrc := chain.NewNetworkServiceRegistryClient(
		nslabelsmatch.NewNetworkServiceRegistryClient(ctx),
		adapters.NetworkServiceServerToClient(mem),
	)

	type test struct {
		name string
		ns   *registry.NetworkService
		want string
	}

	tests := []test{
		{
			name: "CorrectTemplate",
			ns: &registry.NetworkService{
				Matches: []*registry.Match{
					{
						Routes: []*registry.Destination{
							{
								DestinationSelector: map[string]string{
									destinationTestKey: destinationTestCorrectTemplate,
								},
							},
						},
					},
				},
			},
			want: testEnvValue,
		},
		{
			name: "IncorrectTemplate",
			ns: &registry.NetworkService{
				Matches: []*registry.Match{
					{
						Routes: []*registry.Destination{
							{
								DestinationSelector: map[string]string{
									destinationTestKey: destinationTestIncorrectTemplate,
								},
							},
						},
					},
				},
			},
			want: "",
		},
		{
			name: "NotATemplate",
			ns: &registry.NetworkService{
				Matches: []*registry.Match{
					{
						Routes: []*registry.Destination{
							{
								DestinationSelector: map[string]string{
									destinationTestKey: destinationTestNotATemplate,
								},
							},
						},
					},
				},
			},
			want: destinationTestNotATemplate,
		},
	}

	for _, test := range tests {
		tc := test
		t.Run(tc.name, func(t *testing.T) {
			ns, err := nsrc.Register(ctx, tc.ns)
			require.NoError(t, err)

			nsrfc, err := nsrc.Find(ctx, &registry.NetworkServiceQuery{NetworkService: ns})
			require.NoError(t, err)

			ns, err = nsrfc.Recv()
			require.NoError(t, err)

			require.Equal(t, tc.want, ns.Matches[0].Routes[0].DestinationSelector[destinationTestKey])
		})
	}
}
