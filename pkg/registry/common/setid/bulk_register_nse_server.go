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

package setid

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/pkg/errors"
)

type networkServiceRegistryBulkRegisterNSEServer struct {
	registry.NetworkServiceRegistry_BulkRegisterNSEServer
}

func (s *networkServiceRegistryBulkRegisterNSEServer) Send(r *registry.NSERegistration) error {
	if r.NetworkServiceEndpoint.Name == "" {
		if r.NetworkService.Name == "" {
			return errors.New("network service has empty name")
		}
		r.NetworkServiceEndpoint.Name = fmt.Sprintf("%v-%v", r.NetworkService.Name, uuid.New().String())
	}
	return s.NetworkServiceRegistry_BulkRegisterNSEServer.Send(r)
}
