// Copyright (c) 2023 Doc.ai and/or its affiliates.
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

package debug

import (
	"context"

	"github.com/networkservicemesh/sdk/pkg/registry/utils/metadata"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/log/logruslogger"
)

type contextKeyType string

const (
	loggedType                  string         = "registry"
	nsClientRecvLoggedKey       contextKeyType = "nsClientRecvLoggedKey"
	nsClientRecvErrorLoggedKey  contextKeyType = "nsClientRecvErrorLoggedKey"
	nsServerSendLoggedKey       contextKeyType = "nsServerSendLoggedKey"
	nsServerSendErrorLoggedKey  contextKeyType = "nsServerSendErrorLoggedKey"
	nseClientRecvLoggedKey      contextKeyType = "nseClientRecvLoggedKey"
	nseClientRecvErrorLoggedKey contextKeyType = "nseClientRecvErrorLoggedKey"
	nseServerSendLoggedKey      contextKeyType = "nseServerSendLoggedKey"
	nseServerSendErrorLoggedKey contextKeyType = "nseServerSendErrorLoggedKey"
)

// withLog - provides corresponding logger in context
func withLog(parent context.Context) (c context.Context) {
	if parent == nil {
		panic("cannot create context from nil parent")
	}

	fields := []*log.Field{log.NewField("type", loggedType)}
	lLogger := logruslogger.LoggerWithFields(fields)

	return log.WithLog(parent, lLogger)
}

func storeNSClientRecvLogged(ctx context.Context) (stored bool) {
	_, ok := metadata.Map(ctx, true).Load(nsClientRecvLoggedKey)
	if !ok {
		metadata.Map(ctx, true).Store(nsClientRecvLoggedKey, true)
	}
	return !ok
}

func storeNSClientRecvErrorLogged(ctx context.Context) (stored bool) {
	_, ok := metadata.Map(ctx, true).Load(nsClientRecvErrorLoggedKey)
	if !ok {
		metadata.Map(ctx, true).Store(nsClientRecvErrorLoggedKey, true)
	}
	return !ok
}

func storeNSServerSendLogged(ctx context.Context) (stored bool) {
	_, ok := metadata.Map(ctx, true).Load(nsServerSendLoggedKey)
	if !ok {
		metadata.Map(ctx, true).Store(nsServerSendLoggedKey, true)
	}
	return !ok
}

func storeNSServerSendErrorLogged(ctx context.Context) (stored bool) {
	_, ok := metadata.Map(ctx, true).Load(nsServerSendErrorLoggedKey)
	if !ok {
		metadata.Map(ctx, true).Store(nsServerSendErrorLoggedKey, true)
	}
	return !ok
}

func storeNSEClientRecvLogged(ctx context.Context) (stored bool) {
	_, ok := metadata.Map(ctx, true).Load(nseClientRecvLoggedKey)
	if !ok {
		metadata.Map(ctx, true).Store(nseClientRecvLoggedKey, true)
	}
	return !ok
}

func storeNSEClientRecvErrorLogged(ctx context.Context) (stored bool) {
	_, ok := metadata.Map(ctx, true).Load(nseClientRecvErrorLoggedKey)
	if !ok {
		metadata.Map(ctx, true).Store(nseClientRecvErrorLoggedKey, true)
	}
	return !ok
}

func storeNSEServerSendLogged(ctx context.Context) (stored bool) {
	_, ok := metadata.Map(ctx, true).Load(nseServerSendLoggedKey)
	if !ok {
		metadata.Map(ctx, true).Store(nseServerSendLoggedKey, true)
	}
	return !ok
}

func storeNSEServerSendErrorLogged(ctx context.Context) (stored bool) {
	_, ok := metadata.Map(ctx, true).Load(nseServerSendErrorLoggedKey)
	if !ok {
		metadata.Map(ctx, true).Store(nseServerSendErrorLoggedKey, true)
	}
	return !ok
}

func isReadyForLogging(ctx context.Context, isClient bool) (isReady bool) {
	defer func() {
		if r := recover(); r != nil {
			isReady = false
		}
	}()
	metadata.Map(ctx, false)
	return true
}
