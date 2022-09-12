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

package updatepath

import (
	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
)

/*
Logic for Update path:

 0. if Index == 0, and there is no current segment will add one.
 1. If current path segment.Name is equal to segmentName passed, it will just update current connection.Id and exit.
 2. If current path segment.Name is not equal to segmentName:
    2.0 if current path segment.Id is not equal to current connection.Id, will return error.
    2.1 if path has next segment available, but next name is not equal to segmentName, will update both next name and connection.Id.
    2.2 if no next path segment available, it will add one more path segment and generate new Id, update connection.Id.
    2.3 if path has next segment available and next name is segmentName, take Id from next path segment.
*/
func updatePath(conn *networkservice.Connection, segmentName string) (*networkservice.Connection, uint32, error) {
	if conn == nil {
		return nil, 0, errors.New("updatePath cannot be called with a nil conn")
	}
	if conn.Path == nil {
		// If we don't have a Path, add one
		conn.Path = &networkservice.Path{}
	}
	if len(conn.GetPath().GetPathSegments()) == 0 {
		// 0. Index == 0, and there is no current segment
		conn.Path.Index = 0
		if conn.Id == "" {
			// Generate new ID for connection and segment.
			conn.Id = uuid.New().String()
		}
		// Add current segment to list
		conn.Path.PathSegments = append(conn.Path.PathSegments, &networkservice.PathSegment{
			Name: segmentName,
			Id:   conn.Id,
		})
		return conn, 0, nil
	}
	path := conn.GetPath()

	if int(path.Index) < len(path.PathSegments) && path.PathSegments[path.Index].Name == segmentName {
		// 1. current path segment.Name is equal to segmentName
		conn.Id = path.PathSegments[path.Index].Id
		return conn, path.Index, nil
	}

	// 2. current path segment.Name is not equal to segmentName

	// We need to move to next item
	nextIndex := int(path.Index) + 1
	if nextIndex > len(path.GetPathSegments()) {
		// We have index > segments count
		return nil, 0, errors.Errorf("Path.Index+1==%d should be less or equal len(Path.PathSegments)==%d",
			nextIndex, len(path.GetPathSegments()))
	}

	if path.PathSegments[path.Index].Id != conn.Id {
		// 2.0 current path segment.Id is not equal to current connection.Id
		return nil, 0, errors.Errorf("Current PathSegment.Id==%v should be equal Connection.Id==%v",
			path.PathSegments[path.Index].Id, conn.Id)
	}

	if nextIndex < len(path.GetPathSegments()) && path.GetPathSegments()[nextIndex].Name != segmentName {
		// 2.1 path has next segment available, but next name is not equal to segmentName
		path.PathSegments[nextIndex].Name = segmentName
		path.PathSegments[nextIndex].Id = uuid.New().String()
	}

	// Increment index to be accurate to current chain element
	conn.Path.Index++

	if int(conn.Path.Index) >= len(path.GetPathSegments()) {
		// 2.2 no next path segment available
		conn.Id = uuid.New().String()
		path.PathSegments = append(path.PathSegments, &networkservice.PathSegment{
			Name: segmentName,
			Id:   conn.Id,
		})
	} else {
		// 2.3 path has next segment available and next name is segmentName
		conn.Id = path.GetPathSegments()[conn.Path.Index].Id
	}

	return conn, conn.Path.Index - 1, nil
}
