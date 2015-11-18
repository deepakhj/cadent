// Copyright (C) 2015 Myfitnesspal

// testing that our hasherrings matches the same hashring from the graphite world
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
	CH.NumberOfReplicas = 100 //graphite
	CH.SetHasherByName("md5")
	CH.SetElterByName("graphite")
	for _, srv := range srvs {
		CH.Add(srv)
	}
	circ := CH.Circle()
	for _, ss := range CH.Sorted() {
		fmt.Printf("(%d, %s)\n", ss, circ[ss])
	}
	for _, word := range words {
		ss, _ := CH.Get(word)
		fmt.Printf("%s, %s\n", word, ss)

	}
}
