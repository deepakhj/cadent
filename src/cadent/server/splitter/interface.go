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
An interface that is the "runner" for various Line Processors
*/

package splitter

import "time"

type Phase int

const (
	Parsed            Phase = iota // initial parsing on direct incoming
	AccumulatedParsed              // if Accumulated and parsed, so we know NOT to run the accumualtor again
	Sent                           // delivered to output queue
)

type Origin int

const (
	TCP Origin = iota
	UDP
	HTTP
	Other
)

type SplitItem interface {
	Key() []byte
	HasTime() bool
	Timestamp() time.Time
	Line() []byte
	Fields() [][]byte
	Tags() [][][]byte
	Phase() Phase
	SetPhase(Phase)
	Origin() Origin
	SetOrigin(Origin)
	OriginName() string
	SetOriginName(string)
	IsValid() bool
}

type Splitter interface {
	ProcessLine(line []byte) (SplitItem, error)
	Name() (name string)
}
