// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
//
// Copyright (c) 2022 Cisco and/or its affiliates.
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

package fs_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime/debug"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/sdk/pkg/tools/fs"
)

func Test_WatchFile(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	root := t.TempDir()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	filePath := filepath.Join(root, "file1.txt")

	ch := fs.WatchFile(ctx, filePath)

	readEvent := func() []byte {
		select {
		case <-ctx.Done():
			debug.PrintStack()
			t.Fatal("timeout waiting for event", filePath)
		case event := <-ch:
			return event
		}
		return nil
	}

	require.Nil(t, readEvent(), filePath) // Initial file read. nil because file doesn't exist yet

	// We can't use things like os.WriteFile here
	// because MacOS doesn't support recursive directory watching: https://github.com/fsnotify/fsnotify/issues/11
	// and without it event watching for "create and write" operations is not stable,
	// we should have separate "create" and "write" events for consistent behavior.
	f, err := os.OpenFile(filepath.Clean(filePath), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	require.NoError(t, err)
	require.NotNil(t, readEvent(), filePath) // file created

	_, err = f.WriteString("data")
	require.NoError(t, err)
	err = f.Close()
	require.NoError(t, err)
	require.NotNil(t, readEvent(), filePath) // file write

	err = os.RemoveAll(root)
	require.NoError(t, err)
	require.Nil(t, readEvent(), filePath) // file removed

	// Removing file is async operation.
	// Waiting for events should theoretically sync us with the filesystem,
	// but apparently sometimes it's not enough, so MkdirAll can fail because the folder is still locked by the remove operation.
	// Particularly, this can be observed on slow Windows systems.
	// Ideally we would only use require.Eventually, but require.Eventually waits specified tick duration before first check,
	// while os.MkdirAll usually succeeds instantly, therefore explicit call before require.Eventually makes test run faster.
	if os.MkdirAll(root, os.ModePerm) != nil {
		require.Eventually(t, func() bool {
			return os.MkdirAll(root, os.ModePerm) == nil
		}, time.Millisecond*300, time.Millisecond*50)
	}

	err = os.WriteFile(filePath, []byte("data"), os.ModePerm)
	require.NoError(t, err)
	require.NotNil(t, readEvent(), filePath) // file created
}
