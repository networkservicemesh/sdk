// Copyright (c) 2020 Cisco Systems, Inc.
//
// Copyright (c) 2021-2023 Doc.ai and/or its affiliates.
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

// Package debug_test has few tests for logs in debug mode
package debug_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/testutil"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

func TestDebugOutput(t *testing.T) {
	// Configure logging
	// Set output to buffer
	var buff bytes.Buffer
	logrus.SetOutput(&buff)
	logrus.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})
	logrus.SetLevel(logrus.DebugLevel)

	// Create a chain with modifying elements
	ch := chain.NewNetworkServiceServer(
		metadata.NewServer(),
		&testutil.LabelChangerFirstServer{},
		&testutil.LabelChangerSecondServer{},
	)

	request := testutil.NewConnection()

	conn, err := ch.Request(context.Background(), request)
	require.NoError(t, err)
	require.NotNil(t, conn)

	e, err := ch.Close(context.Background(), conn)
	require.NoError(t, err)
	require.NotNil(t, e)

	expectedOutput :=
		"\x1b[37m [DEBU] [id:conn-1] [type:networkService] \x1b[0mserver-request={\"connection\":" +
			"{\"id\":\"conn-1\",\"context\":{\"ip_context\":{\"src_ip_required\":true}}},\"mechanism_preferences\":" +
			"[{\"cls\":\"LOCAL\",\"type\":\"KERNEL\"},{\"cls\":\"LOCAL\",\"type\":\"KERNEL\",\"parameters\":{\"label\":\"v2\"}}]}" +
			"\n\x1b[37m [DEBU] [id:conn-1] [type:networkService] \x1b[0mserver-request-response={\"id\":\"conn-1\",\"context\":" +
			"{\"ip_context\":{\"src_ip_required\":true}},\"labels\":{\"Label\":\"B\"}}" +
			"\n\x1b[37m [DEBU] [id:conn-1] [connID:conn-1] [metadata:server] [type:networkService] \x1b[0mmetadata deleted" +
			"\n\x1b[37m [DEBU] [id:conn-1] [type:networkService] \x1b[0mserver-close={\"id\":\"conn-1\",\"context\":{\"ip_context\":" +
			"{\"src_ip_required\":true}},\"labels\":{\"Label\":\"D\"}}" +
			"\n\x1b[37m [DEBU] [id:conn-1] [type:networkService] \x1b[0mserver-close-response={\"id\":\"conn-1\",\"context\":" +
			"{\"ip_context\":{\"src_ip_required\":true}},\"labels\":{\"Label\":\"X\"}}\n"

	result := testutil.TrimLogTime(&buff)
	require.Equal(t, expectedOutput, result)
}

func TestErrorOutput(t *testing.T) {
	// Configure logging
	// Set output to buffer
	var buff bytes.Buffer
	logrus.SetOutput(&buff)

	logrus.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})
	logrus.SetLevel(logrus.DebugLevel)

	// Create a chain with modifying elements
	ch := chain.NewNetworkServiceServer(
		metadata.NewServer(),
		&testutil.LabelChangerFirstServer{},
		&testutil.LabelChangerSecondServer{},
		&testutil.ErrorServer{},
	)

	request := testutil.NewConnection()

	conn, err := ch.Request(context.Background(), request)
	require.Error(t, err)
	require.Nil(t, conn)

	expectedOutput :=
		"\x1b[37m [DEBU] [id:conn-1] [type:networkService] \x1b[0mserver-request={\"connection\":" +
			"{\"id\":\"conn-1\",\"context\":{\"ip_context\":{\"src_ip_required\":true}}},\"mechanism_preferences\":" +
			"[{\"cls\":\"LOCAL\",\"type\":\"KERNEL\"},{\"cls\":\"LOCAL\",\"type\":\"KERNEL\",\"parameters\":{\"label\":\"v2\"}}]}\n" +
			"\x1b[37m [DEBU] [id:conn-1] [type:networkService] \x1b[0mserver-request-response={\"id\":\"conn-1\",\"context\":" +
			"{\"ip_context\":{\"src_ip_required\":true}},\"labels\":{\"Label\":\"B\"}}\n" +
			"\x1b[31m [ERRO] [id:conn-1] [type:networkService] \x1b[0mError returned from sdk/pkg/networkservice/core/testutil/ErrorServer.Request:" +
			" Error returned from api/pkg/api/networkservice/networkServiceClient.Close;" +
			"\tgithub.com/networkservicemesh/sdk/pkg/networkservice/core/trace.(*beginTraceClient).Close;" +
			"\t\t/root/go/pkg/mod/github.com/networkservicemesh/sdk@v0.5.1-0.20210929180427-ec235de055f1/pkg/networkservice/core/trace/client.go:85;" +
			"\tgithub.com/networkservicemesh/sdk/pkg/networkservice/core/next.(*nextClient).Close;" +
			"\t\t/root/go/pkg/mod/github.com/networkservicemesh/sdk@v0.5.1-0.20210929180427-ec235de055f1/pkg/networkservice/core/next/client.go:65;" +
			"\tgithub.com/networkservicemesh/sdk/pkg/networkservice/core/next.(*nextClient).Close;" +
			"\t\t/root/go/pkg/mod/github.com/networkservicemesh/sdk@v0.5.1-0.20210929180427-ec235de055f1/pkg/networkservice/core/next/client.go:65;" +
			"\tgithub.com/networkservicemesh/sdk/pkg/networkservice/core/trace.(*endTraceClient).Close;" +
			"\t\t/root/go/pkg/mod/github.com/networkservicemesh/sdk@v0.5.1-0.20210929180427-ec235de055f1/pkg/networkservice/core/trace/client.go:106;" +
			"\tgithub.com/networkservicemesh/sdk/pkg/networkservice/core/next.(*nextClient).Close;" +
			"\t\t/root/go/pkg/mod/github.com/networkservicemesh/sdk@v0.5.1-0.20210929180427-ec235de055f1/pkg/networkservice/core/next/client.go:65;\t\n" +
			"\x1b[37m [DEBU] [id:conn-1] [connID:conn-1] [metadata:server] [type:networkService] \x1b[0mmetadata deleted\n"

	result := testutil.TrimLogTime(&buff)
	require.Equal(t, expectedOutput, result)
}
