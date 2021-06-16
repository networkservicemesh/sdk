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
	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
)

// OnRestore is an action that should be performed on restore request
type OnRestore int

const (
	// OnRestoreRestore - restore should be performed
	OnRestoreRestore OnRestore = iota
	// OnRestoreHeal - heal should be performed
	OnRestoreHeal
	// OnRestoreIgnore - restore request should be ignored
	OnRestoreIgnore
)

// Option is an option pattern for heal server
type Option func(healOpts *healOptions)

// WithOnHeal - sets client used 'onHeal'.
// * If we detect we need to heal, onHeal.Request is used to heal.
// * If we can't heal connection, onHeal.Close will be called.
// * If onHeal is nil, then we simply set onHeal to this client chain element.
// Since networkservice.NetworkServiceClient is an interface (and thus a pointer) *networkservice.NetworkServiceClient
// is a double pointer. Meaning it points to a place that points to a place that implements
// networkservice.NetworkServiceClient. This is done because when we use heal.NewClient as part of a chain, we may not
// *have* a pointer to this chain.
func WithOnHeal(onHeal *networkservice.NetworkServiceClient) Option {
	return func(healOpts *healOptions) {
		healOpts.onHeal = onHeal
	}
}

// WithOnRestore sets on restore action. Default `OnRestoreRestore`.
// IMPORTANT: should be set `OnRestoreIgnore` for the Forwarder, because it results in NSMgr doesn't understanding that
// Request is coming from Forwarder (https://github.com/networkservicemesh/sdk/issues/970).
func WithOnRestore(onRestore OnRestore) Option {
	return func(healOpts *healOptions) {
		healOpts.onRestore = onRestore
	}
}

// WithRestoreTimeout sets restore timeout. Default `1m`.
func WithRestoreTimeout(restoreTimeout time.Duration) Option {
	return func(healOpts *healOptions) {
		healOpts.restoreTimeout = restoreTimeout
	}
}

type healOptions struct {
	onHeal         *networkservice.NetworkServiceClient
	onRestore      OnRestore
	restoreTimeout time.Duration
}
