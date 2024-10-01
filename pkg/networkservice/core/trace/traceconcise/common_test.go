// Copyright (c) 2023-2024 Cisco and/or its affiliates.
//
// Copyright (c) 2024 Nordix Foundation.
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

// Package traceconcise_test has few tests for logs in concise mode
package traceconcise_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/trace/testutil"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

func TestOutput(t *testing.T) {
	t.Skip()
	// Configure logging
	// Set output to buffer
	var buff bytes.Buffer
	logrus.SetOutput(&buff)
	logrus.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})
	log.EnableTracing(true)

	// Create a chain with modifying elements
	ch := chain.NewNetworkServiceServer(
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
		" [INFO] [id:conn-1] [type:networkService] server-request={\"connection\":" +
			"{\"id\":\"conn-1\",\"context\":{\"ip_context\":{\"src_ip_required\":true}}},\"mechanism_preferences\":" +
			"[{\"cls\":\"LOCAL\",\"type\":\"KERNEL\"},{\"cls\":\"LOCAL\",\"type\":\"KERNEL\",\"parameters\":{\"label\":\"v2\"}}]}" +
			"\n [INFO] [id:conn-1] [type:networkService] server-request-response={\"id\":\"conn-1\",\"context\":" +
			"{\"ip_context\":{\"src_ip_required\":true}},\"labels\":{\"Label\":\"B\"}}" +
			"\n [INFO] [id:conn-1] [type:networkService] server-close={\"id\":\"conn-1\",\"context\":{\"ip_context\":" +
			"{\"src_ip_required\":true}},\"labels\":{\"Label\":\"D\"}}" +
			"\n [INFO] [id:conn-1] [type:networkService] server-close-response={\"id\":\"conn-1\",\"context\":" +
			"{\"ip_context\":{\"src_ip_required\":true}},\"labels\":{\"Label\":\"X\"}}\n"

	result := testutil.TrimLogTime(&buff)
	require.Equal(t, expectedOutput, result)
}

func TestErrorOutput(t *testing.T) {
	t.Skip()
	// Configure logging
	// Set output to buffer
	var buff bytes.Buffer
	logrus.SetOutput(&buff)

	logrus.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})
	log.EnableTracing(true)
	logrus.SetLevel(logrus.InfoLevel)

	// Create a chain with modifying elements
	ch := chain.NewNetworkServiceServer(
		&testutil.LabelChangerFirstServer{},
		&testutil.LabelChangerSecondServer{},
		&testutil.ErrorServer{},
	)

	request := testutil.NewConnection()

	conn, err := ch.Request(context.Background(), request)
	require.Error(t, err)
	require.Nil(t, conn)

	expectedOutput :=
		" [INFO] [id:conn-1] [type:networkService] server-request={\"connection\":" +
			"{\"id\":\"conn-1\",\"context\":{\"ip_context\":{\"src_ip_required\":true}}},\"mechanism_preferences\":" +
			"[{\"cls\":\"LOCAL\",\"type\":\"KERNEL\"},{\"cls\":\"LOCAL\",\"type\":\"KERNEL\",\"parameters\":{\"label\":\"v2\"}}]}\n" +
			" [INFO] [id:conn-1] [type:networkService] server-request-response={\"id\":\"conn-1\",\"context\":" +
			"{\"ip_context\":{\"src_ip_required\":true}},\"labels\":{\"Label\":\"B\"}}\n" +
			" [ERRO] [id:conn-1] [type:networkService] Error returned from sdk/pkg/networkservice/core/trace/testutil/ErrorServer.Request:" +
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
			"\t\t/root/go/pkg/mod/github.com/networkservicemesh/sdk@v0.5.1-0.20210929180427-ec235de055f1/pkg/networkservice/core/next/client.go:65;\t\n"

	result := testutil.TrimLogTime(&buff)
	require.Equal(t, expectedOutput, result)
}
