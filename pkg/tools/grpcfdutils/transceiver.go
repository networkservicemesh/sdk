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

package grpcfdutils

import (
	"net"
	"os"
	"runtime"

	"github.com/edwarnicke/grpcfd"
)

type NotifiableFDTransceiver struct {
	grpcfd.FDTransceiver
	net.Addr

	OnRecvFile map[string]func()
}

func (w *NotifiableFDTransceiver) RecvFileByURL(urlStr string) (<-chan *os.File, error) {
	recv, err := w.FDTransceiver.RecvFileByURL(urlStr)
	if err != nil {
		return nil, err
	}

	var fileCh = make(chan *os.File)
	go func() {
		for f := range recv {
			runtime.SetFinalizer(f, func(file *os.File) {
				onFileClosedFunc, ok := w.OnRecvFile[urlStr]
				if ok {
					onFileClosedFunc()
				}
			})
			fileCh <- f
		}
	}()

	return fileCh, nil
}
