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

package spire

import (
	"context"
	"os"
	"strings"

	"github.com/spiffe/go-spiffe/v2/svid/x509svid"
	"github.com/spiffe/go-spiffe/v2/workloadapi"

	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

// FetchX509Source fetches X509 source via default spire unix path. Uses svid picker based on executable.
func FetchX509Source(ctx context.Context) (*workloadapi.X509Source, error) {
	executable, _ := os.Executable()
	return workloadapi.NewX509Source(ctx,
		workloadapi.WithClientOptions(
			workloadapi.WithLogger(log.Entry(ctx)),
			workloadapi.WithAddr("unix://run/spire/sockets/agent.sock")),
		workloadapi.WithDefaultX509SVIDPicker(func(svids []*x509svid.SVID) *x509svid.SVID {
			if len(svids) == 0 {
				return nil
			}
			for _, svid := range svids {
				if strings.HasSuffix(svid.ID.Path(), executable) {
					return svid
				}
			}
			return svids[0]
		}))
}
