// Copyright (c) 2020 Cisco and/or its affiliates.
//
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

package metadata

import (
	"context"
	"sync"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
)

type metaDataKey struct{}

func store(parent context.Context, id string, mdMap *metaDataMap) context.Context {
	if _, ok := parent.Value(metaDataKey{}).(*metaData); !ok {
		md, _ := mdMap.LoadOrStore(id, &metaData{})
		return context.WithValue(parent, metaDataKey{}, md)
	}
	return parent
}

func del(parent context.Context, id string, mdMap *metaDataMap) context.Context {
	if _, ok := parent.Value(metaDataKey{}).(*metaData); !ok {
		if md, ok := mdMap.LoadAndDelete(id); ok {
			return context.WithValue(parent, metaDataKey{}, md)
		}
		return context.WithValue(parent, metaDataKey{}, new(metaData))
	}
	return parent
}

// Map - Return the client (or server) per Connection.Id metadata *sync.Map
func Map(parent context.Context, isClient bool) *sync.Map {
	m, ok := parent.Value(metaDataKey{}).(*metaData)
	if !ok || m == nil {
		panic("please add metadata chain element to your chain")
	}
	if isClient {
		return &m.client
	}
	return &m.server
}

// IsClient - returns true if in implements networkservice.NetworkServiceClient
func IsClient(in interface{}) bool {
	_, ok := in.(networkservice.NetworkServiceClient)
	return ok
}
