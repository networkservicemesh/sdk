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

// Package discovercrossnse TODO
package discovercrossnse

import (
	"context"

	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

type networkserviceNameKey struct{}

type crossConnectEndpointName struct{}

func loadNetworkServiceEndpointName(ctx context.Context) string {
	v, ok := metadata.Map(ctx, false).Load(networkserviceNameKey{})
	if !ok {
		return ""
	}
	return v.(string)
}

func storeNetworkServiceEndpointName(ctx context.Context, v string) {
	metadata.Map(ctx, false).Store(networkserviceNameKey{}, v)
}

func loadCrossConnectEndpointName(ctx context.Context) string {
	v, ok := metadata.Map(ctx, false).Load(crossConnectEndpointName{})
	if !ok {
		return ""
	}
	return v.(string)
}

func storeCrossConnectEndpointName(ctx context.Context, v string) {
	metadata.Map(ctx, false).Store(crossConnectEndpointName{}, v)
}
