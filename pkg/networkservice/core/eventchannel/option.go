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

// Package eventchannel provides API for creating monitoring components  via golang channels
package eventchannel

// MonitorConnectionServerOption applies specific parameters for MonitorConnectionServer.
type MonitorConnectionServerOption interface {
	apply(s *monitorConnectionServer)
}

type monitorConnectionServerOptionFunc func(*monitorConnectionServer)

func (f monitorConnectionServerOptionFunc) apply(s *monitorConnectionServer) {
	f(s)
}

// WithConnectChannel adds for MonitorConnectionServer an option for monitoring connection count.
func WithConnectChannel(connectCh chan<- int) MonitorConnectionServerOption {
	return monitorConnectionServerOptionFunc(func(s *monitorConnectionServer) {
		s.connectCh = connectCh
	})
}
