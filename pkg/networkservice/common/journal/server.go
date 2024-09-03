// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
//
// Copyright (c) 2023 Cisco and/or its affiliates.
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

// Package journal emits IP and PATH related event messages to NATS.
// The journal may be used for healing IPAM and/or auditing
// connection activity.
package journal

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	stan "github.com/nats-io/stan.go"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/clock"
)

// ActionRequest indicates that the event seen is a connection request.
// The connection requests may also be a keep alive.
const ActionRequest = "request"

// ActionClose indicates that the event captured is a connection close.
const ActionClose = "close"

// Entry is populated and serialized to NATS.
type Entry struct {
	Time         time.Time
	Sources      []string
	Destinations []string
	Action       string
	Path         *networkservice.Path
}

type journalServer struct {
	journalID string
	nats      stan.Conn
}

func (srv *journalServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	clockTime := clock.FromContext(ctx)

	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		return conn, err
	}

	src := conn.GetContext().GetIpContext().GetSrcIpAddrs()
	dst := conn.GetContext().GetIpContext().GetDstIpAddrs()
	path := conn.GetPath()

	entry := Entry{
		Time:         clockTime.Now().UTC(),
		Sources:      src,
		Destinations: dst,
		Action:       ActionRequest,
		Path:         path,
	}

	err = srv.publish(&entry)

	return conn, err
}

func (srv *journalServer) Close(ctx context.Context, connection *networkservice.Connection) (*empty.Empty, error) {
	clockTime := clock.FromContext(ctx)

	src := connection.GetContext().GetIpContext().GetSrcIpAddrs()
	dst := connection.GetContext().GetIpContext().GetDstIpAddrs()

	entry := Entry{
		Time:         clockTime.Now().UTC(),
		Sources:      src,
		Destinations: dst,
		Action:       ActionClose,
	}

	// squash error if present
	_ = srv.publish(&entry)

	return next.Server(ctx).Close(ctx, connection)
}

func (srv *journalServer) publish(entry *Entry) error {
	js, err := json.Marshal(entry)
	if err != nil {
		return errors.Wrapf(err, "failed to get JSON of %s", entry)
	}

	err = srv.nats.Publish(srv.journalID, js)
	if err != nil {
		return errors.Wrapf(err, "failed to publish %s to the cluster %s", js, srv.journalID)
	}
	return nil
}

// NewServer creates a new journaling server with the name journalID using provided streaming NATS connection.
func NewServer(journalID string, stanConn stan.Conn) (networkservice.NetworkServiceServer, error) {
	if strings.TrimSpace(journalID) == "" {
		return nil, errors.New("journal id is nil")
	}
	return &journalServer{
		journalID: journalID,
		nats:      stanConn,
	}, nil
}
