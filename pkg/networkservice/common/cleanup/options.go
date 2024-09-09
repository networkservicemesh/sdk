// Copyright (c) 2022 Cisco and/or its affiliates.
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

package cleanup

type options struct {
	ccClose bool
	doneCh  chan struct{}
}

// Option - options for the cleanup chain element.
type Option func(*options)

// WithoutGRPCCall - closes client connection to prevent calling requests/closes on other endpoints.
func WithoutGRPCCall() Option {
	return func(o *options) {
		o.ccClose = true
	}
}

// WithDoneChan - receives a channel to notify the end of cleaning.
func WithDoneChan(doneCh chan struct{}) Option {
	return func(o *options) {
		o.doneCh = doneCh
	}
}
