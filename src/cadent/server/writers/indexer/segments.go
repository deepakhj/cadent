/*
Copyright 2016 Under Armour, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

/*
	Segment parsing for inserting
*/

package indexer

import (
	"strings"
)

/** Being Cassandra we need some mappings to match the schemas **/

/*
 	CREATE TYPE metric.segment_pos (
    		pos int,
    		segment text
	);
*/
type PathSegment struct {
	Pos     int
	Segment string
}

/*
 	CREATE TABLE metric.path (
            segment frozen<segment_pos>,
            length int,
            path text,
            id varchar,
            has_data boolean,
            PRIMARY KEY (segment, length, path, id)
        )
*/
type PathPart struct {
	Segment PathSegment
	Path    string
	Id      string
	Length  int
	Hasdata bool
}

type ParsedPath struct {
	Segments []PathSegment
	Paths    []PathPart
	Len      int
	Parts    []string
}

func NewParsedPath(name string, uid string) *ParsedPath {
	p := new(ParsedPath)
	p.Paths = make([]PathPart, 0)
	p.Segments = make([]PathSegment, 0)

	p.Parts = strings.Split(name, ".")
	p.Len = len(p.Parts)

	cur_part := ""

	for idx, part := range p.Parts {
		if len(cur_part) > 1 {
			cur_part += "."
		}
		cur_part += part
		on_segment := PathSegment{
			Segment: cur_part,
			Pos:     idx,
		}
		p.Segments = append(p.Segments, on_segment)

		on_path := PathPart{
			Id:      uid,
			Segment: on_segment,
			Path:    name,
			Length:  p.Len - 1, // starts at 0
		}

		p.Paths = append(p.Paths, on_path)
	}
	return p
}

func (p *ParsedPath) Last() string {
	return p.Parts[p.Len-1]
}
