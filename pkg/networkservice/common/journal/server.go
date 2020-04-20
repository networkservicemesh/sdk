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
//
// Package journal emits IP and PATH related event messages to NATS.
// The journal may be used for healing IPAM and/or auditing
// connection activity.
package journal

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	stan "github.com/nats-io/stan.go"
	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

// ActionRequest indicates that the event seen is a connection request.
// The connection requests may also be a keep alive.
const ActionRequest = "request"

// ActionClose indicates that the event captured is a connection close.
const ActionClose = "close"

// Entry is populated and serialized to NATS.
type Entry struct {
	Time        time.Time
	Source      string
	Destination string
	Action      string
	Path        *networkservice.Path
}

type journalServer struct {
	journalID string
	nats      stan.Conn
}

func (srv journalServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		return conn, err
	}

	src := conn.GetContext().GetIpContext().GetSrcIpAddr()
	dst := conn.GetContext().GetIpContext().GetDstIpAddr()
	path := conn.GetPath()

	entry := Entry{
		Time:        time.Now().UTC(),
		Source:      src,
		Destination: dst,
		Action:      ActionRequest,
		Path:        path,
	}

	err = srv.publish(&entry)

	return conn, err
}
func (srv journalServer) Close(ctx context.Context, connection *networkservice.Connection) (*empty.Empty, error) {
	src := connection.GetContext().GetIpContext().GetSrcIpAddr()
	dst := connection.GetContext().GetIpContext().GetDstIpAddr()
	entry := Entry{
		Time:        time.Now().UTC(),
		Source:      src,
		Destination: dst,
		Action:      ActionClose,
	}

	// squash error if present
	_ = srv.publish(&entry)

	return next.Server(ctx).Close(ctx, connection)
}

func (srv journalServer) publish(entry *Entry) error {
	js, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	err = srv.nats.Publish(srv.journalID, js)
	if err != nil {
		return err
	}
	return nil
}

// NewServer creates a new journaling server with the name journalID using provided streaming NATS connection
func NewServer(journalID string, stanConn stan.Conn) (networkservice.NetworkServiceServer, error) {
	if strings.TrimSpace(journalID) == "" {
		return nil, errors.New("journal id is nil")
	}
	return &journalServer{
		journalID: journalID,
		nats:      stanConn,
	}, nil
}
