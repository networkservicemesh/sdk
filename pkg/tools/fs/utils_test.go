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

package fs_test

import (
	"context"
	iofs "io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/tools/fs"
)

func checkPathError(t *testing.T, err error) {
	if err == nil {
		return
	}

	if ferr, ok := err.(*iofs.PathError); ok {
		if u := ferr.Unwrap(); u != nil {
			t.Log(u)
		}
	}
	require.Nil(t, err)
}

func Test_WatchFile(t *testing.T) {
	root := filepath.Join(os.TempDir(), t.Name())

	path := filepath.Join(root, uuid.New().String())
	err := os.MkdirAll(path, os.ModePerm)
	defer func() {
		_ = os.RemoveAll(path)
	}()
	checkPathError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	filePath := filepath.Join(path, "file1.txt")
	ch := fs.WatchFile(ctx, filePath)

	expectEvent := func() []byte {
		select {
		case <-ctx.Done():
			debug.PrintStack()
			require.Fail(t, "timeout waiting for message from", filePath)
		case event := <-ch:
			return event
		}
		return nil
	}

	require.Nil(t, expectEvent(), filePath) // initial file read, nil because file doesn't exist

	err = ioutil.WriteFile(filePath, []byte("data"), os.ModePerm)
	checkPathError(t, err)
	require.NotNil(t, expectEvent(), filePath) // file created

	macOSName := "darwin"
	// https://github.com/fsnotify/fsnotify/issues/11
	if runtime.GOOS != macOSName {
		require.NotNil(t, expectEvent(), filePath) // file write. MacOS doesn't support write events
	}
	time.Sleep(time.Millisecond * 50)

	err = os.RemoveAll(path)
	checkPathError(t, err)
	if runtime.GOOS != macOSName {
		require.Nil(t, expectEvent(), filePath) // file removed
	} else {
		event := expectEvent()
		if event != nil { // ...but MacOS sometimes sends write events. Very rarely, about 1/500 test retries
			event = expectEvent()
		}
		require.Nil(t, event, filePath)
	}

	err = os.MkdirAll(path, os.ModePerm)
	for i := 0; i < 5 && err != nil; i++ {
		// Removing file is async operation.
		// Waiting for events should theoretically sync us with filesystem,
		// but apparently sometimes it's not enough, so MkdirAll can fail because the folder is still locked by remove.
		// Particularly, this can be observed on slow Windows OS systems.
		time.Sleep(time.Millisecond * 50)
		err = os.MkdirAll(path, os.ModePerm)
	}
	checkPathError(t, err)

	err = ioutil.WriteFile(filePath, []byte("data"), os.ModePerm)
	checkPathError(t, err)
	require.NotNil(t, expectEvent(), filePath) // file created
}
