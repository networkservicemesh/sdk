// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
//
// Copyright (c) 2022 Cisco Systems, Inc.
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

// Package updatepath provides a chain element that sets the id of an incoming or outgoing request
package updatepath

import (
	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk/pkg/registry/common/grpcmetadata"
)

/*
Logic for Update path:

 0. if Index == 0, and there is no current segment will add one.
 1. If current path segment.Name is equal to segmentName, it will just exit.
 2. If current path segment.Name is not equal to segmentName:
    2.0 if path has next segment available, but next name is not equal to segmentName, will update next name.
    2.1 if no next path segment available, it will add one more path segment and generate new Id.
*/
func updatePath(path *grpcmetadata.Path, segmentName string) (*grpcmetadata.Path, uint32, error) {
	if path == nil {
		return nil, 0, errors.New("updatePath cannot be called with a nil path")
	}
	if len(path.PathSegments) == 0 {
		// 0. Index == 0, and there is no current segment
		path.Index = 0
		// Add current segment to list
		path.PathSegments = append(path.PathSegments, &grpcmetadata.PathSegment{
			Name: segmentName,
			ID:   uuid.New().String(),
		})
		return path, 0, nil
	}

	// 1. current path segment.Name is equal to segmentName
	if int(path.Index) < len(path.PathSegments) && path.PathSegments[path.Index].Name == segmentName {
		return path, path.Index, nil
	}

	// 2. current path segment.Name is not equal to segmentName

	// We need to move to next item
	nextIndex := int(path.Index) + 1
	if nextIndex > len(path.PathSegments) {
		// We have index > segments count
		return nil, 0, errors.Errorf("Path.Index+1==%d should be less or equal len(Path.PathSegments)==%d",
			nextIndex, len(path.PathSegments))
	}

	if nextIndex < len(path.PathSegments) && path.PathSegments[nextIndex].Name != segmentName {
		// 2.0 path has next segment available, but next name is not equal to segmentName
		path.PathSegments[nextIndex].Name = segmentName
		path.PathSegments[nextIndex].ID = uuid.New().String()
	}

	// Increment index to be accurate to current chain element
	path.Index++

	if int(path.Index) >= len(path.PathSegments) {
		// 2.1 no next path segment available
		path.PathSegments = append(path.PathSegments, &grpcmetadata.PathSegment{
			Name: segmentName,
			ID:   uuid.New().String(),
		})
	}

	return path, path.Index - 1, nil
}
