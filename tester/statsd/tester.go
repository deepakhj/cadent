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

// Copyright (C) 2015 Myfitnesspal

// testing that our hasherrings matches the same hashring from the node-js world (statsd)
//
package main

import (
	"../../../consistent"
	"fmt"
	"strings"
)

func main() {

	srvs := []string{"s1", "s2", "s3", "s4", "s5", "s6"}
	parg := "A written look gloves the here lyric. How does a separated helmet chalk? The minister intervenes across the beautiful bliss. The thankful equilibrium gangs your stationary apple."
	words := strings.Split(parg, " ")

	CH := consistent.New()
	CH.NumberOfReplicas = 40 //statsd hashing default
	CH.SetHasherByName("nodejs-hashring")
	CH.SetElterByName("statsd")
	for _, srv := range srvs {
		CH.Add(srv)
	}
	circ := CH.Circle()
	for _, ss := range CH.Sorted() {
		fmt.Printf("(%d, %s)\n", ss, circ[ss])
	}
	return
	for _, word := range words {
		ss, _ := CH.Get(word)
		fmt.Printf("%s, %s\n", word, ss)

	}
}
