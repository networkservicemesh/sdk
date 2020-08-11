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

// Package fswatcher provide common tools for FS watching
package fswatcher

import (
	"context"

	"github.com/fsnotify/fsnotify"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/trace"
)

// WatchOn watches for events on given ...paths
func WatchOn(ctx context.Context, onEvent func(event fsnotify.Event) error, paths ...string) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer func() { _ = watcher.Close() }()

	for _, path := range paths {
		if err := watcher.Add(path); err != nil {
			return err
		}
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event := <-watcher.Events:
			if err := onEvent(event); err != nil {
				return err
			}
		case err := <-watcher.Errors:
			trace.Log(ctx).Errorf("Watch %+v, error: %v", paths, err.Error())
			return err
		}
	}
}
