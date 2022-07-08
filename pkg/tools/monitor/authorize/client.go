// Copyright (c) 2022 Doc.ai and/or its affiliates.
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

package authorize

import (
	"context"
	"fmt"

	"github.com/golang-jwt/jwt/v4"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/sirupsen/logrus"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/tools/monitor/next"
	"github.com/networkservicemesh/sdk/pkg/tools/opa"
)

type authorizeMonitorConnectionsClient struct {
	policies policiesList
}

// NewMonitorConnectionsClient - returns a new authorization NewMonitorConnectionsClient
// Authorize client checks rigiht side of path.
func NewMonitorConnectionsClient(opts ...Option) networkservice.MonitorConnectionClient {
	var result = &authorizeMonitorConnectionsClient{
		policies: []Policy{
			opa.WithServiceOwnConnectionPolicy(),
		},
	}

	for _, o := range opts {
		o.apply(&result.policies)
	}

	return result
}

func (a *authorizeMonitorConnectionsClient) MonitorConnections(ctx context.Context, in *networkservice.MonitorScopeSelector, opts ...grpc.CallOption) (networkservice.MonitorConnection_MonitorConnectionsClient, error) {
	clt, err := next.MonitorConnectionClient(ctx).MonitorConnections(ctx, in, opts...)
	if err != nil {
		return nil, err
	}
	event, err := clt.Recv()
	if err != nil {
		return nil, err
	}
	// add var connIdMap map[spiffeid.ID]*networkservice.Connection
	for k, v := range event.Connections {
		logrus.Infof("Conn %s: %v \n", k, v)
		nextToken := v.GetNextPathSegment().Token
		logrus.Infof("Conn next path token  %v \n", nextToken)
		tokenDecoded, err := jwt.ParseWithClaims(
			nextToken, &jwt.RegisteredClaims{},
			func(token *jwt.Token) (interface{}, error) {return []byte("AllYourBase"), nil},
		)
		if err != nil {
			return nil, fmt.Errorf("error decoding connection token: %+v", err) 
		}
		logrus.Infof("decoded Token %v \n", tokenDecoded)
		decodedClaims := tokenDecoded.Claims.(jwt.RegisteredClaims)
		logrus.Infof("decoded Claims %v \n", decodedClaims)
		logrus.Infof("Subject from the Claims %v", decodedClaims.Subject)
		logrus.Infof("Audienct from the Claims %v", decodedClaims.Audience)
	}
	
	source, err := workloadapi.NewX509Source(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting x509 source: %+v", err) 
	}
	svid, err := source.GetX509SVID()
	if err != nil {
		return nil, fmt.Errorf("error getting x509 svid: %+v", err) 
	}
	logrus.Infof("Service Own Spiffe ID %v", svid.ID)

	// if err := a.policies.check(ctx, svid); err != nil {
	// 	return nil, err
	// }

	return clt, nil
}
