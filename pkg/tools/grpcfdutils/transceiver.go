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

// +build linux

// Package grpcfdutils provides utilities for grpcfd library
package grpcfdutils

import (
	"context"
	"net"
	"os"

	"github.com/edwarnicke/grpcfd"
	"github.com/pkg/errors"
	"google.golang.org/grpc/peer"
)

// notifiableFDTransceiver - grpcfd.Transceiver wrapper checking that received FDs are closed
//     onRecvFile - callback receiving inodeURL string and a file received by grpcfd
type notifiableFDTransceiver struct {
	grpcfd.FDTransceiver
	net.Addr

	onRecvFile func(string, *os.File)
}

// RecvFileByURL - wrapper of grpcfd.FDRecver method invoking callback when a file is received by grpcfd
func (w *notifiableFDTransceiver) RecvFileByURL(urlStr string) (<-chan *os.File, error) {
	recv, err := w.FDTransceiver.RecvFileByURL(urlStr)
	if err != nil {
		return nil, err
	}

	var fileCh = make(chan *os.File)
	go func() {
		for f := range recv {
			w.onRecvFile(urlStr, f)
			fileCh <- f
		}
	}()

	return fileCh, nil
}

// InjectOnFileReceivedCallback - injects callback into grpcfd.FDTransceiver that will be invoked on file received
func InjectOnFileReceivedCallback(ctx context.Context, callback func(string, *os.File)) error {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return errors.Errorf("No peer in context")
	}

	transceiver, ok := p.Addr.(grpcfd.FDTransceiver)
	if !ok {
		return errors.Errorf("No grpcfd transceiver in context")
	}

	p.Addr = &notifiableFDTransceiver{
		FDTransceiver: transceiver,
		Addr:          p.Addr,
		onRecvFile:    callback,
	}

	return nil
}
