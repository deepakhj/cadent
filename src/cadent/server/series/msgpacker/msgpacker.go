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

// need:  go get -u github.com/tinylib/msgp

//go:generate msgp -o msgpacker.msgp.go --file msgpacker.go

package msgpacker

// sort-hand keys for space purposes
type FullStat struct {
	Time  int64   `json:"t"  codec:"t" msg:"t"`
	Min   float64 `json:"n"  codec:"n" msg:"n"`
	Max   float64 `json:"m"  codec:"m" msg:"m"`
	Sum   float64 `json:"s"  codec:"s" msg:"s"`
	Last  float64 `json:"l"  codec:"l" msg:"l"`
	Count int64   `json:"c"  codec:"c" msg:"c"`
}

type StatSmall struct {
	Time int64   `json:"t" codec:"t" msg:"t"`
	Val  float64 `json:"v" codec:"v" msg:"v"`
}

type Stat struct {
	StatType  bool       `json:"t" codec:"t" msg:"t"`
	Stat      *FullStat  `json:"s" codec:"s" msg:"s"`
	SmallStat *StatSmall `json:"m" codec:"m" msg:"m"`
}

type Stats struct {
	FullTimeResolution bool    `json:"r" codec:"r" msg:"r"`
	Stats              []*Stat `json:"s" codec:"s" msg:"s"`
}
