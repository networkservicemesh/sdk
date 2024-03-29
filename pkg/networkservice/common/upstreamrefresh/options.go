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

package upstreamrefresh

type options struct {
	localNotifier *notifier
}

// Option - option for upstreamrefresh chain element
type Option func(o *options)

// WithLocalNotifications - allows all connections to receive events, if at least one of them received an event from upstream.
func WithLocalNotifications() Option {
	return func(o *options) {
		o.localNotifier = newNotifier()
	}
}
