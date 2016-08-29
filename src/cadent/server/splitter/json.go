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
   a raw json splitter.  which is not really a "splitter" but just a data struct around
   an incoming json blob
*/

package splitter

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

/****************** RUNNERS *********************/
const JSON_SPLIT_NAME = "json"

var errBadJsonLine = errors.New("Invalid Json object")

// a json input handler format (like that of openTSDB
/* {
metric: xxx,
value: 123,
timestamp: 123123,
tags: {
 moo: goo,
 foo: bar
 }
}
*/

type JsonStructSplitItem struct {
	Metric string            `json:"metric"`
	Value  float64           `json:"value"`
	Time   int64             `json:"timestamp"`
	Tags   map[string]string `json:"tags,omitempty"`
}

type JsonSplitItem struct {
	inkey    []byte
	inline   []byte
	intime   time.Time
	infields [][]byte
	inphase  Phase
	inorigin Origin
	inoname  string
	tags     [][][]byte
}

func (g *JsonSplitItem) Key() []byte {
	return g.inkey
}

func (g *JsonSplitItem) Tags() [][][]byte {
	return g.tags
}

func (g *JsonSplitItem) HasTime() bool {
	return true
}

func (g *JsonSplitItem) Timestamp() time.Time {
	return g.intime
}

func (g *JsonSplitItem) Line() []byte {
	return g.inline
}

func (g *JsonSplitItem) Fields() [][]byte {
	return g.infields
}

func (g *JsonSplitItem) Phase() Phase {
	return g.inphase
}

func (g *JsonSplitItem) SetPhase(n Phase) {
	g.inphase = n
}

func (g *JsonSplitItem) Origin() Origin {
	return g.inorigin
}

func (g *JsonSplitItem) SetOrigin(n Origin) {
	g.inorigin = n
}

func (g *JsonSplitItem) OriginName() string {
	return g.inoname
}

func (g *JsonSplitItem) SetOriginName(n string) {
	g.inoname = n
}

func (g *JsonSplitItem) IsValid() bool {
	return len(g.inkey) > 0
}

func (g *JsonSplitItem) String() string {
	return fmt.Sprintf("Splitter: Json: %s @ %d", string(g.Key()), g.intime.Unix())
}

// splitter

type JsonSplitter struct{}

func (g *JsonSplitter) Name() (name string) { return JSON_SPLIT_NAME }

func NewJsonSplitter(conf map[string]interface{}) (*JsonSplitter, error) {
	job := &JsonSplitter{}
	return job, nil
}

func (g *JsonSplitter) ParseJson(j_item *JsonStructSplitItem) (SplitItem, error) {
	if len(j_item.Metric) == 0 {
		return nil, errBadJsonLine
	}

	spl_item := new(JsonSplitItem)
	spl_item.inkey = []byte(j_item.Metric)

	// nano or second tstamps
	if j_item.Time > 2147483647 {
		spl_item.intime = time.Unix(0, j_item.Time)
	} else {
		spl_item.intime = time.Unix(j_item.Time, 0)
	}
	spl_item.infields = append(spl_item.infields, spl_item.inkey)
	spl_item.infields = append(spl_item.infields, []byte(fmt.Sprintf("%v", j_item.Value)))
	spl_item.infields = append(spl_item.infields, []byte(fmt.Sprintf("%v", j_item.Time)))

	if len(j_item.Tags) > 0 {

		otags := make([][][]byte, 0)
		for nm, val := range j_item.Tags {
			otags = append(otags, [][]byte{[]byte(nm), []byte(val)})
		}

		spl_item.tags = otags
	}

	// need to "squeeze" the air out of the incoming
	var err error
	spl_item.inline, err = json.Marshal(j_item)
	return spl_item, err
}

// array parser
func (g *JsonSplitter) ProcessLines(line []byte) ([]SplitItem, error) {
	var j_items []JsonStructSplitItem
	err := json.Unmarshal(line, j_items)
	if err != nil {
		return nil, err
	}

	if len(j_items) == 0 {
		return nil, errBadJsonLine
	}

	var jsonitems []SplitItem

	for _, item := range j_items {
		j, err := g.ParseJson(&item)
		if err != nil {
			return jsonitems, err
		}
		jsonitems = append(jsonitems, j)
	}
	return jsonitems, nil
}

func (g *JsonSplitter) ProcessLine(line []byte) (SplitItem, error) {
	j_item := new(JsonStructSplitItem)
	err := json.Unmarshal(line, j_item)
	if err != nil {
		return nil, err
	}

	return g.ParseJson(j_item)

}
