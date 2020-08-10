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

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/trace"
	"github.com/networkservicemesh/sdk/pkg/tools/fswatcher"
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

func watchFile(ctx context.Context, path string, onChanged func([]byte)) error {
	watcher, err := fswatcher.WatchOn(path)
	if err != nil {
		return err
	}
	defer func() { _ = watcher.Close() }()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-watcher.Events:
			data, err := ioutil.ReadFile(filepath.Clean(path))
			if err != nil {
				trace.Log(ctx).Errorf("An error during read file %v, error: %v", path, err.Error())
				return err
			}
			onChanged(data)
		case err := <-watcher.Errors:
			trace.Log(ctx).Errorf("Watch %v, error: %v", path, err.Error())
			return err
		}
	}
}
