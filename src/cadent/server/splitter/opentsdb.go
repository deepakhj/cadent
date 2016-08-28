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
   OpenTSDB line format is almost exactly the same as the extended graphite format
   except there is a "put" as the first word .. the rest is the same so we just strip the put part

   put <key> <val> <time> <tag> <tag>


*/

package splitter

import (
	"bytes"
	"errors"
)

/****************** RUNNERS *********************/
const OPENTSDB_NAME = "opentsdb"

var OPENTSDB_CMD = []byte("put ")

var errBadOpenTSDBLine = errors.New("Invalid OpenTSDB line")

type OpenTSDBSplitItem struct {
	GraphiteSplitItem
}

type OpenTSDBSplitter struct {
}

func (g *OpenTSDBSplitter) Name() (name string) { return OPENTSDB_NAME }

func NewOpenTSDBSplitter(conf map[string]interface{}) (*OpenTSDBSplitter, error) {

	//<key> <value> <time> <thigns>
	job := &OpenTSDBSplitter{}
	return job, nil
}

func (g *OpenTSDBSplitter) ProcessLine(line []byte) (SplitItem, error) {
	//put <key> <value> <time> <more> <more>
	if len(line) < 4 {
		return nil, errBadOpenTSDBLine
	}
	if bytes.Equal(OPENTSDB_CMD, line[0:4]) {
		g_fmat := new(GraphiteSplitter)
		g_fmat.key_index=0
		g_fmat.time_index=2
		return g_fmat.ProcessLine(line[4:])

	}
	return nil, errBadOpenTSDBLine

}
