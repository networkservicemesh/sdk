// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
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

package localbypass

import (
	"errors"
	"github.com/networkservicemesh/api/pkg/api/registry"
)

type localBypassNSEFindClient struct {
	url string
	registry.NetworkServiceEndpointRegistry_FindClient
}

func (s *localBypassNSEFindClient) Recv() (*registry.NetworkServiceEndpoint, error) {
	nse, err := s.NetworkServiceEndpointRegistry_FindClient.Recv()
	if err != nil {
		return nse, err
	}
	if nse.Url == s.url {
		return nil, errors.New("NSMgr found unregistered endpoint")
	}
	return nse, err
}
