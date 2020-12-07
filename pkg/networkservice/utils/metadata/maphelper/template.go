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

package maphelper

import (
	"context"
	"sync"

	"github.com/cheekybits/genny/generic"

	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

type prefix generic.Type
type keyType generic.Type
type valueType generic.Type

type _prefixMapKeyType keyType

var _valueTypeNil valueType

type prefixMapMetaDataHelper struct {
	m *sync.Map
}

func prefixMapMetaData(ctx context.Context, isClient bool) *prefixMapMetaDataHelper {
	return &prefixMapMetaDataHelper{
		m: metadata.Map(ctx, isClient),
	}
}

func (h *prefixMapMetaDataHelper) Store(key keyType, value valueType) {
	h.m.Store(_prefixMapKeyType(key), value)
}

func (h *prefixMapMetaDataHelper) LoadOrStore(key keyType, value valueType) (valueType, bool) {
	raw, ok := h.m.LoadOrStore(_prefixMapKeyType(key), value)
	return raw.(valueType), ok
}

func (h *prefixMapMetaDataHelper) Load(key keyType) (valueType, bool) {
	if raw, ok := h.m.Load(_prefixMapKeyType(key)); ok {
		return raw.(valueType), true
	}
	return _valueTypeNil, false
}

func (h *prefixMapMetaDataHelper) LoadAndDelete(key keyType) (valueType, bool) {
	if raw, ok := h.m.LoadAndDelete(_prefixMapKeyType(key)); ok {
		return raw.(valueType), true
	}
	return _valueTypeNil, false
}

func (h *prefixMapMetaDataHelper) Delete(key keyType) {
	h.m.Delete(_prefixMapKeyType(key))
}
