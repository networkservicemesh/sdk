// Copyright (c) 2023 Cisco and/or its affiliates.
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

package authorize

import (
	"context"

	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

type key struct{}

// store sets a flag stored per Connection.Id metadata.
// It is used to keep a successful Request.
// Based on this, we can understand whether the Request is a refresh.
func store(ctx context.Context, isClient bool) {
	metadata.Map(ctx, isClient).Store(key{}, struct{}{})
}

// load returns a flag stored per Connection.Id metadata.
// It is used to determine a refresh.
func load(ctx context.Context, isClient bool) (ok bool) {
	_, ok = metadata.Map(ctx, isClient).Load(key{})
	return
}

// del deletes a flag stored per Connection.Id metadata.
func del(ctx context.Context, isClient bool) {
	metadata.Map(ctx, isClient).Delete(key{})
}
