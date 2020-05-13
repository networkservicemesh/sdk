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

package dnscontext

const (
	// SearchProperty means search list for host-name lookup
	SearchProperty = "search"
	// NameserverProperty means name server IP address
	NameserverProperty = "nameserver"
	// OptionsProperty  allows certain internal resolver variables to be modified
	OptionsProperty = "options"
	// AnyDomain means that allowed any host-name
	AnyDomain              = "."
	defaultPlugin          = "forward"
	conflictResolverPlugin = "fanout"
	serverBlockTemplate    = `%v {
	%v . %v
	%v
}`
)
