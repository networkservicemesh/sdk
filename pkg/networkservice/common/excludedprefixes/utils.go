// Copyright (c) 2020 Doc.ai and/or its affiliates.
//
// Copyright (c) 2020 Cisco and/or its affiliates.
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

package excludedprefixes

import (
	"context"
	"io/ioutil"
	"path/filepath"

	"github.com/fsnotify/fsnotify"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/trace"
)

func removeDuplicates(elements []string) []string {
	encountered := map[string]bool{}
	result := []string{}

	for index := range elements {
		if encountered[elements[index]] {
			continue
		}
		encountered[elements[index]] = true
		result = append(result, elements[index])
	}
	return result
}

func watchFile(ctx context.Context, directoryPath, filePath string, onChanged func([]byte)) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	if err := watcher.Add(directoryPath); err != nil {
		return err
	}

	defer func() {
		_ = watcher.Close()
	}()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case e := <-watcher.Events:
			if !(e.Name == filePath ||
				e.Op&fsnotify.Create == fsnotify.Create ||
				e.Op&fsnotify.Write == fsnotify.Write) {
				continue
			}

			data, err := ioutil.ReadFile(filepath.Clean(filePath))
			if err != nil {
				trace.Log(ctx).Errorf("An error during read file %v, error: %v", filePath, err.Error())
				return err
			}
			onChanged(data)
		case err := <-watcher.Errors:
			trace.Log(ctx).Errorf("Watch %v, error: %v", filePath, err.Error())
			return err
		}
	}
}
