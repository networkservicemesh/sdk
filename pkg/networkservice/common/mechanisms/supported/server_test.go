// Copyright (c) 2021 Cisco and/or its affiliates.
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

package supported_test

import (
	"context"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/memif"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vxlan"
	"github.com/stretchr/testify/assert"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/supported"
)

var testInput = []struct {
	supported   []string
	unsupported []string
}{
	{
		supported: []string{
			kernel.MECHANISM, memif.MECHANISM,
		},
		unsupported: []string{vxlan.MECHANISM},
	},
}

func TestSupportedPreferences(t *testing.T) {
	for _, input := range testInput {
		server := supported.NewServer(supported.WithSupportedMechanismTypes(input.supported...))
		request := &networkservice.NetworkServiceRequest{}
		for _, supported := range input.supported {
			request.MechanismPreferences = append(request.GetMechanismPreferences(), &networkservice.Mechanism{
				Type: supported,
			})
		}
		_, err := server.Request(context.Background(), request)
		assert.NoError(t, err)
		_, err = server.Close(context.Background(), request.Connection)
		assert.NoError(t, err)
	}
}

func TestUnSupportedPreferences(t *testing.T) {
	for _, input := range testInput {
		server := supported.NewServer(supported.WithSupportedMechanismTypes(input.supported...))
		request := &networkservice.NetworkServiceRequest{}
		for _, unsupported := range input.unsupported {
			request.MechanismPreferences = append(request.GetMechanismPreferences(), &networkservice.Mechanism{
				Type: unsupported,
			})
		}
		_, err := server.Request(context.Background(), request)
		assert.Error(t, err)
		_, err = server.Close(context.Background(), request.Connection)
		assert.NoError(t, err)
	}
}

func TestSupportedMechanism(t *testing.T) {
	for _, input := range testInput {
		server := supported.NewServer(supported.WithSupportedMechanismTypes(input.supported...))
		for _, supported := range input.supported {
			request := &networkservice.NetworkServiceRequest{
				Connection: &networkservice.Connection{
					Mechanism: &networkservice.Mechanism{
						Type: supported,
					},
				},
			}
			_, err := server.Request(context.Background(), request)
			assert.NoError(t, err)
			_, err = server.Close(context.Background(), request.Connection)
			assert.NoError(t, err)
		}
	}
}

func TestUnSupportedMechanism(t *testing.T) {
	for _, input := range testInput {
		server := supported.NewServer(supported.WithSupportedMechanismTypes(input.supported...))
		for _, unsupported := range input.unsupported {
			request := &networkservice.NetworkServiceRequest{
				Connection: &networkservice.Connection{
					Mechanism: &networkservice.Mechanism{
						Type: unsupported,
					},
				},
			}
			_, err := server.Request(context.Background(), request)
			assert.Error(t, err)
			_, err = server.Close(context.Background(), request.Connection)
			assert.NoError(t, err)
		}
	}
}
