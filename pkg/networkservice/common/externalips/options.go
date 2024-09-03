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

package externalips

// Option option for externalips server.
type Option func(*externalIPsServer)

// WithFilePath means listen file by passed path.
func WithFilePath(p string) Option {
	return func(server *externalIPsServer) {
		server.updateCh = monitorMapFromFile(server.chainCtx, p)
	}
}

// WithUpdateChannel passed to server specific channel for listening updates.
func WithUpdateChannel(ch <-chan map[string]string) Option {
	return func(server *externalIPsServer) {
		server.updateCh = ch
	}
}
