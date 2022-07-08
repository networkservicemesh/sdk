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

// Package authorize provides authz checks for incoming or returning connections.
package authorize

import (
	"fmt"

	"github.com/golang-jwt/jwt/v4"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/sirupsen/logrus"
	"github.com/spiffe/go-spiffe/v2/workloadapi"

	"github.com/networkservicemesh/sdk/pkg/tools/monitor/next"
	"github.com/networkservicemesh/sdk/pkg/tools/opa"
)

type authorizeServer struct {
	policies policiesList
}

// NewMonitorConnectionsServer - returns a new authorization networkservicemesh.MonitorConnectionServer
// Authorize server checks left side of Path.
func NewMonitorConnectionsServer(opts ...Option) networkservice.MonitorConnectionServer {
	// todo: add policies
	var s = &authorizeServer{
		policies: []Policy{
			opa.WithServiceOwnConnectionPolicy(),
		},
	}
	for _, o := range opts {
		o.apply(&s.policies)
	}
	return s
}

func (a *authorizeServer) MonitorConnections(in *networkservice.MonitorScopeSelector, srv networkservice.MonitorConnection_MonitorConnectionsServer) error {
	ctx := srv.Context()
	var decodedClaims jwt.RegisteredClaims
	for _, seg := range in.GetPathSegments(){
		logrus.Printf("Conn next path token  %v \n", seg.Token)
		tokenDecoded, err := jwt.ParseWithClaims(
			seg.Token, &jwt.RegisteredClaims{},
			func(token *jwt.Token) (interface{}, error) {return []byte("AllYourBase"), nil},
		)
		if err != nil {
			return fmt.Errorf("error decoding connection token: %+v", err)
		}
		logrus.Infof("decoded Token %v \n", tokenDecoded)
		decodedClaims = tokenDecoded.Claims.(jwt.RegisteredClaims)
		logrus.Infof("decoded Claims %v \n", decodedClaims)
		logrus.Infof("Subject from the Claims %v", decodedClaims.Subject)
		logrus.Infof("Audienct from the Claims %v", decodedClaims.Audience)
	
	}
	source, err := workloadapi.NewX509Source(ctx)
	if err != nil {
		return fmt.Errorf("error getting x509 source: %+v", err)
	}
	svid, err := source.GetX509SVID()
	if err != nil {
		return fmt.Errorf("error getting x509 svid: %+v", err)
	}
	logrus.Infof("Service Own Spiffe ID %v", svid.ID)


	// if _, ok := peer.FromContext(ctx); ok {
	// 	if err := a.policies.check(ctx, svid); err != nil {
	// 		return err
	// 	}
	// }
	return next.MonitorConnectionServer(ctx).MonitorConnections(in, srv)
}
