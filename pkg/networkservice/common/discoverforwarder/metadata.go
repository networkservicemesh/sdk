// Copyright (c) 2022 Doc.ai and/or its affiliates.
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

package discoverforwarder

import (
	"context"

	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

type selectedForworderKey struct{}

func loadForwarderName(ctx context.Context) string {
	v, ok := metadata.Map(ctx, false).Load(selectedForworderKey{})
	if !ok {
		return ""
	}
	return v.(string)
}

func storeForwarderName(ctx context.Context, v string) {
	metadata.Map(ctx, false).Store(selectedForworderKey{}, v)
}
