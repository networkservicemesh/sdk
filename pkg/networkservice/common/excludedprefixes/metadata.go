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

package excludedprefixes

import (
	"context"

	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

type keyType struct{}

type prefixesInfo struct {
	previousFilePrefixes   []string
	previousClientPrefixes []string
	previousDiff           []string
}

func load(ctx context.Context) (prefixesInfo, bool) {
	val, ok := metadata.Map(ctx, false).Load(keyType{})
	if !ok {
		return prefixesInfo{}, false
	}

	return val.(prefixesInfo), true
}

func store(ctx context.Context, info prefixesInfo) {
	metadata.Map(ctx, false).Store(keyType{}, info)
}

func del(ctx context.Context) {
	metadata.Map(ctx, false).Delete(keyType{})
}
