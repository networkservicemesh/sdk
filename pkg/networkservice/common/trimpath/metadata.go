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

package trimpath

import (
	"context"

	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

type key struct{}

// dontTrimPath stores the fact we should not trim the path in in per Connection.Id metadata.
// should only be called in trimPathClient.
func dontTrimPath(ctx context.Context) {
	metadata.Map(ctx, true).Store(key{}, key{})
}

// shouldTrimPath returns true if the server should trim the path, false otherwise.
func shouldTrimPath(ctx context.Context) (ok bool) {
	m := metadata.Map(ctx, true)
	_, ok = m.LoadAndDelete(key{})
	return !ok
}
