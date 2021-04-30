// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
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

package fs

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/networkservicemesh/sdk/pkg/tools/log"

	"github.com/fsnotify/fsnotify"
)

// WatchFile watches file changes even if the watching file does not exist or removed.
// Sends nil value in the channel on file removing.
// Closes channel due to unexpected os error or context is done.
func WatchFile(ctx context.Context, filePath string) <-chan []byte {
	result := make(chan []byte)
	logger := log.FromContext(ctx).WithField("fs.WatchFile", filePath)

	watcher, err := fsnotify.NewWatcher()

	if err != nil {
		logger.Errorf("can not create node poller: %v", err.Error())
		_ = watcher.Close()
		close(result)
		return result
	}

	directoryPath, _ := filepath.Split(filePath)
	if directoryPath != "" {
		if _, err := os.Stat(directoryPath); os.IsNotExist(err) {
			err = os.MkdirAll(directoryPath, os.ModePerm)
			if err != nil {
				logger.Errorf("can not create directory: %v", err.Error())
				_ = watcher.Close()
				close(result)
				return result
			}
		}
	}

	if err := watcher.Add(directoryPath); err != nil {
		logger.Errorf("an error during add a directory \"%v\": %v", directoryPath, err.Error())
		_ = watcher.Close()
		close(result)
		return result
	}

	go func() {
		defer func() {
			_ = watcher.Close()
		}()
		monitorFile(ctx, filePath, watcher, result)
	}()
	return result
}

func monitorFile(ctx context.Context, filePath string, watcher *fsnotify.Watcher, notifyCh chan<- []byte) {
	logger := log.FromContext(ctx).WithField("fs.monitorFile", filePath)

	bytes, _ := ioutil.ReadFile(filepath.Clean(filePath))
	if !sendOrClose(ctx, notifyCh, bytes) {
		return
	}

	for {
		select {
		case <-ctx.Done():
			logger.Error(ctx.Err().Error())
			close(notifyCh)
			return
		case e := <-watcher.Events:
			if !strings.HasSuffix(filePath, filepath.Clean(e.Name)) {
				continue
			}
			if e.Op&(fsnotify.Remove|fsnotify.Rename) > 0 {
				logger.Warn("Removed")
				if !sendOrClose(ctx, notifyCh, nil) {
					return
				}
			} else if e.Op&(fsnotify.Write|fsnotify.Create) == 0 {
				continue
			}
			data, err := ioutil.ReadFile(filepath.Clean(filePath))
			for err != nil && ctx.Err() == nil {
				time.Sleep(time.Millisecond * 50)
				logger.Warn(err.Error())
				data, err = ioutil.ReadFile(filepath.Clean(filePath))
			}
			if !sendOrClose(ctx, notifyCh, data) {
				return
			}
		case err := <-watcher.Errors:
			if err != nil {
				logger.Error(err.Error())
				close(notifyCh)
				return
			}
		}
	}
}

func sendOrClose(ctx context.Context, notifyCh chan<- []byte, data []byte) bool {
	select {
	case notifyCh <- data:
		return true
	case <-ctx.Done():
		close(notifyCh)
		return false
	}
}
