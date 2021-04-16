// Copyright (c) 2020 Cisco and/or its affiliates.
//
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
	"context"
	"sync"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
)

type connectionInfo struct {
	mut                   sync.Mutex
	conn                  *networkservice.Connection
	state                 connectionState
	successVerificationCh chan struct{}
}

//go:generate go-syncmap -output connection_info_map.gen.go -type connectionInfoMap<string,*connectionInfo>

type connectionInfoMap sync.Map

type ctxWrapper struct {
	mut     sync.Mutex
	request *networkservice.NetworkServiceRequest
	ctx     context.Context
	cancel  func()
}

//go:generate go-syncmap -output context_wrapper_map.gen.go -type ctxWrapperMap<string,*ctxWrapper>

type ctxWrapperMap sync.Map
