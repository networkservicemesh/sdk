// Copyright (c) 2020 Doc.ai and/or its affiliates.
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

package metadatahelper

import (
	"context"
	"sync"

	"github.com/cheekybits/genny/generic"

	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

type prefix generic.Type
type valueType generic.Type

type _prefixKeyType struct{}

type prefixMetadataHelper struct {
	m *sync.Map
}

func prefixMetadata(ctx context.Context, isClient bool) *prefixMetadataHelper {
	return &prefixMetadataHelper{
		m: metadata.Map(ctx, isClient),
	}
}

func (h *prefixMetadataHelper) Store(value valueType) {
	h.m.Store(_prefixKeyType{}, value)
}

func (h *prefixMetadataHelper) LoadOrStore(value valueType) (valueType, bool) {
	raw, ok := h.m.LoadOrStore(_prefixKeyType{}, value)
	return raw.(valueType), ok
}

func (h *prefixMetadataHelper) Load() (valueType, bool) {
	if raw, ok := h.m.Load(_prefixKeyType{}); ok {
		return raw.(valueType), true
	}
	return func() (v valueType) { return }(), false
}

func (h *prefixMetadataHelper) LoadAndDelete() (valueType, bool) {
	if raw, ok := h.m.LoadAndDelete(_prefixKeyType{}); ok {
		return raw.(valueType), true
	}
	return func() (v valueType) { return }(), false
}

func (h *prefixMetadataHelper) Delete() {
	h.m.Delete(_prefixKeyType{})
}
