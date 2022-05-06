// Copyright (c) 2020-2022 Cisco and/or its affiliates.
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
)

type entry struct {
	spiffeID string
	selector string
}

type federatedEntry struct {
	entry
	federatesWith string
}

type option struct {
	ctx        context.Context
	agentID    string
	agentConf  string
	serverConf string
	spireRoot  string
	entries    []*entry
	fEntries   []*federatedEntry
}

// Option for spire
type Option func(*option)

// WithContext - use ctx as context for starting spire
func WithContext(ctx context.Context) Option {
	return func(o *option) {
		o.ctx = ctx
	}
}

// WithAgentID - agentID for starting spire
func WithAgentID(agentID string) Option {
	return func(o *option) {
		o.agentID = agentID
	}
}

// WithEntry - Option to add Entry to spire-server.  May be used multiple times.
func WithEntry(spiffeID, selector string) Option {
	return func(o *option) {
		o.entries = append(o.entries, &entry{
			spiffeID: spiffeID,
			selector: selector,
		})
	}
}

// WithFederatedEntry - Option to add federated Entry to spire-server.  May be used multiple times.
func WithFederatedEntry(spiffeID, selector, federatesWith string) Option {
	return func(o *option) {
		o.fEntries = append(o.fEntries, &federatedEntry{
			entry:         entry{spiffeID: spiffeID, selector: selector},
			federatesWith: federatesWith,
		})
	}
}

// WithAgentConfig - adds agent config
func WithAgentConfig(conf string) Option {
	return func(o *option) {
		o.agentConf = conf
	}
}

// WithServerConfig - adds server config
func WithServerConfig(conf string) Option {
	return func(o *option) {
		o.serverConf = conf
	}
}

// WithRoot - sets root folder
func WithRoot(root string) Option {
	return func(o *option) {
		o.spireRoot = root
	}
}
