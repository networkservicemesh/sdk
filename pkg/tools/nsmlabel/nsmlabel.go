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

// Package nsmlabel provides helper function for buildning networkservice.NetworkServiceClient
package nsmlabel

import (
	"net/url"
	"strings"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/common"
)

// Parse parses url.URL to tuple of required networkservice.Request parameters
func Parse(u *url.URL) (mechanism *networkservice.Mechanism, networkService string, labels map[string]string) {
	labels = make(map[string]string)
	mechanism = &networkservice.Mechanism{Cls: cls.LOCAL, Type: u.Scheme}
	for k, values := range u.Query() {
		labels[k] = strings.Join(values, ",")
	}
	networkService = u.Host

	segments := strings.Split(u.Path, "/")

	if len(segments) > 1 {
		mechanism.Parameters = make(map[string]string)
		mechanism.Parameters[common.InterfaceNameKey] = segments[len(segments)-1]
	}

	return mechanism, networkService, labels
}
