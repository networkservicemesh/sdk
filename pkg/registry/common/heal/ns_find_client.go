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

package heal

import (
	"context"

	"github.com/networkservicemesh/api/pkg/api/registry"
)

type healNSFindClient struct {
	ctx          context.Context
	err          error
	createStream func() (registry.NetworkServiceRegistry_FindClient, error)

	registry.NetworkServiceRegistry_FindClient
}

func (c *healNSFindClient) Recv() (*registry.NetworkServiceResponse, error) {
	if c.err != nil {
		return nil, c.err
	}

	nsResp, err := c.NetworkServiceRegistry_FindClient.Recv()
	for ; err != nil; nsResp, err = c.NetworkServiceRegistry_FindClient.Recv() {
		c.NetworkServiceRegistry_FindClient, err = c.createStream()
		for ; err != nil; c.NetworkServiceRegistry_FindClient, err = c.createStream() {
			if c.ctx.Err() != nil {
				c.err = c.ctx.Err()
				return nil, c.err
			}
		}

		if c.ctx.Err() != nil {
			c.err = c.ctx.Err()
			return nil, c.err
		}
	}

	return nsResp, nil
}
