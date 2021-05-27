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

package connect

import (
	"io"

	"github.com/networkservicemesh/api/pkg/api/registry"
)

type connectNSFindServer struct {
	client    *nsClient
	clientURL string
	err       error

	*connectNSServer
	registry.NetworkServiceRegistry_FindServer
}

func (s *connectNSFindServer) Send(ns *registry.NetworkService) error {
	if s.err != nil {
		return s.err
	}

	switch err := s.NetworkServiceRegistry_FindServer.Send(ns); {
	case err == io.EOF:
		s.err = err
		s.closeClient(s.client, s.clientURL)
		return io.EOF
	case err != nil:
		if s.client.client.ctx.Err() != nil {
			s.err = err
			s.deleteClient(s.client, s.clientURL)
		}
		return err
	}

	return nil
}
