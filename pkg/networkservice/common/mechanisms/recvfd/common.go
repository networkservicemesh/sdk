// Copyright (c) 2020-2022 Cisco and/or its affiliates.
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

//go:build linux
// +build linux

package recvfd

import (
	"context"
	"net/url"
	"os"

	"github.com/edwarnicke/grpcfd"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/common"
	"github.com/pkg/errors"
)

func recvFDAndSwapInodeToFile(ctx context.Context, fileMap *perConnectionFileMap, parameters map[string]string, recv grpcfd.FDRecver) error {
	// Get the inodeURL from  parameters
	inodeURLStr, ok := parameters[common.InodeURL]
	if !ok {
		return nil
	}

	// Transform string to URL for correctness checking and ease of use
	inodeURL, err := url.Parse(inodeURLStr)
	if err != nil {
		return errors.WithStack(err)
	}

	// Is it an inode?
	if inodeURL.Scheme != "inode" {
		return nil
	}

	<-fileMap.executor.AsyncExec(func() {
		// Look to see if we already have a file for that inode
		file, ok := fileMap.filesByInodeURL[inodeURLStr]
		if !ok {
			// If we don't have a file for that inode... get it from the grpcfd.FDRecver
			var fileCh <-chan *os.File
			fileCh, err = recv.RecvFileByURL(inodeURLStr)
			if err != nil {
				err = errors.WithStack(err)
				return
			}
			// Wait for the file to arrive on the fileCh or the context to expire
			select {
			case <-ctx.Done():
				err = errors.Wrapf(ctx.Err(), "timeout in recvfd waiting for %s", inodeURLStr)
				return
			case file = <-fileCh:
				// If we get the file, remember it in the fileMap so we can reuse it later
				// Note: This is done because we want to present a single consistent filename to
				// any of the other chain elements using the information, and since that filename will be
				// file:///proc/${pid}/fd/${fd} we need to remember it because each time we get it from the
				// grpcfd.Recver it will be a *different* fd and thus a different filename
				if file == nil {
					err = errors.Wrapf(ctx.Err(), "nil file received for %s", inodeURLStr)
					return
				}
				fileMap.filesByInodeURL[inodeURL.String()] = file
			}
		}

		// Swap out the inodeURL for a fileURL in the parameters
		fileURL := &url.URL{Scheme: "file", Path: file.Name()}
		parameters[common.InodeURL] = fileURL.String()

		// Remember the swap so we can undo it later
		fileMap.inodeURLbyFilename[file.Name()] = inodeURL
	})
	return err
}

func swapFileToInode(fileMap *perConnectionFileMap, parameters map[string]string) error {
	// Get the inodeURL from  parameters
	fileURLStr, ok := parameters[common.InodeURL]
	if !ok {
		return nil
	}

	// Transform string to URL for correctness checking and ease of use
	fileURL, err := url.Parse(fileURLStr)
	if err != nil {
		return errors.WithStack(err)
	}

	// Is it a file?
	if fileURL.Scheme != "file" {
		return nil
	}
	<-fileMap.executor.AsyncExec(func() {
		// Do we have an inodeURL to translate it back to?
		inodeURL, ok := fileMap.inodeURLbyFilename[fileURL.Path]
		if !ok {
			return
		}
		// Swap the fileURL for the inodeURL in parameters
		parameters[common.InodeURL] = inodeURL.String()

		// This is used to clean up files sent by MechanismPreferences that were *not* selected to be the
		// connection mechanism
		for inodeURLStr, file := range fileMap.filesByInodeURL {
			if inodeURLStr != inodeURL.String() {
				delete(fileMap.filesByInodeURL, inodeURLStr)
				_ = file.Close()
			}
		}
	})
	return nil
}
