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

package dnsresolve_test

import (
	"context"
	"errors"
	"fmt"
	"net"
)

type testResolver struct {
	srvRecords  map[string][]*net.SRV
	hostRecords map[string][]net.IPAddr
}

func (t *testResolver) LookupSRV(ctx context.Context, service, proto, name string) (string, []*net.SRV, error) {
	key := fmt.Sprintf("_%v._%v.%v", service, proto, name)
	result := t.srvRecords[key]
	if len(result) == 0 {
		return "", nil, errors.New("can not resolve " + key)
	}
	return "", result, nil
}

func (t *testResolver) LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error) {
	result := t.hostRecords[host]
	if len(result) == 0 {
		return nil, errors.New("can not resolve " + host)
	}
	return result, nil
}
