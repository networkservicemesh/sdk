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

	"github.com/networkservicemesh/sdk/pkg/tools/stringurl"
)

type localBypassNSEFindServer struct {
	nseURLs  *stringurl.Map
	nsmgrUrl string

	registry.NetworkServiceEndpointRegistry_FindServer
}

func (s *localBypassNSEFindServer) Send(endpoint *registry.NetworkServiceEndpoint) error {
	if u, ok := s.nseURLs.Load(endpoint.Name); ok {
		endpoint.Url = u.String()
	}
	if endpoint.Url == s.nsmgrUrl {
		return errors.New("NSMgr found unregistered endpoint")
	}

	return s.NetworkServiceEndpointRegistry_FindServer.Send(endpoint)
}
