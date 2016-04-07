// Copyright (C) 2015 Myfitnesspal

// returns the "string" representation of the server based on the Vnodes and name
// Different elts will yield different hash rings for the same hashing function
// this just let's us "imitate" other hashrings out there
//

package consistent

import (
	"strconv"
	"strings"
)

// note we need to remove the "udp|tcp|http://" for this to be accurate
func cleanElt(elt string) string {
	return strings.Replace(strings.Replace(strings.Replace(elt, "udp://", "", -1), "tcp://", "", -1), "http://", "", -1)
}

//see  https://github.com/graphite-project/graphite-web/blob/910085aa12ef51d6de78d8e3ef929a3503615b30/webapp/graphite/render/hashing.py
// note we need to remove the "udp|tcp://" for this to be accurate
func graphiteElt(elt string, idx int) string {
	return cleanElt(elt) + ":" + strconv.Itoa(idx)
}

//see https://github.com/3rd-Eden/node-hashring/blob/master/index.js
// around line 119
// note we need to remove the "udp|tcp://" for this to be accurate
func statsdElt(elt string, idx int) string {
	return cleanElt(elt) + "-" + strconv.Itoa(idx)
}

//see  https://github.com/stathat/consistent/blob/master/consistent.go
// line 68 or so
func consistentElt(elt string, idx int) string {
	return strconv.Itoa(idx) + elt
}
