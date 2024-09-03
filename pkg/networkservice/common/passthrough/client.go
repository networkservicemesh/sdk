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

// Package passthrough combines chain elements used in passthrough cases
package passthrough

import (
	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/passthrough/replacelabels"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/passthrough/replacensename"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
)

// NewClient returns a combination for passthrough cases.
func NewClient(labels map[string]string) networkservice.NetworkServiceClient {
	return chain.NewNetworkServiceClient(
		replacelabels.NewClient(labels),
		replacensename.NewClient(),
	)
}
