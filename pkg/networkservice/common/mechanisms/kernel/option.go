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

package kernel

import "github.com/RoaringBitmap/roaring"

type options struct {
	interfaceName string
	vlans         *roaring.Bitmap
}

// Option is an option pattern for kernelMechanismClient/Server
type Option func(o *options)

// WithInterfaceName sets interface name
func WithInterfaceName(interfaceName string) Option {
	return func(o *options) {
		o.interfaceName = limitName(interfaceName)
	}
}

// WithVLAN sets assignable vlan id range
func WithVLAN() Option {
	return func(o *options) {
		o.vlans = roaring.New()
		o.vlans.AddRange(1, 4095)
	}
}
