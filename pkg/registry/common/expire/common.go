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

package expire

import (
	"time"

	"github.com/networkservicemesh/api/pkg/api/registry"
)

const defaultPeriod = time.Second * 5

func defaultNowFunc() func() int64 {
	return func() int64 {
		return time.Now().Unix()
	}
}

func getExpiredNSEs(nseMap map[string]*registry.NetworkServiceEndpoint, now int64) []*registry.NetworkServiceEndpoint {
	var list []*registry.NetworkServiceEndpoint
	for _, v := range nseMap {
		if v.ExpirationTime == nil {
			continue
		}
		if now >= v.ExpirationTime.Seconds {
			list = append(list, v)
		}
	}
	return list
}
