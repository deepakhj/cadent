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
	Utils for basic the key inputs for converting glob patters to something
	golang understands
*/

package indexer

import (
	"cadent/server/repr"
	"fmt"
	"regexp"
	"strings"
)

// need regex?
func needRegex(metric string) bool {
	return strings.IndexAny(metric, "*?[{^$") >= 0
}

func regifyKeyString(name string) string {
	regable := strings.Replace(name, "..", ".", -1)
	regable = strings.Replace(regable, "{", "(", -1)
	regable = strings.Replace(regable, "}", ")", -1)
	regable = strings.Replace(regable, ",", "|", -1)
	regable = strings.Replace(regable, ".", "\\.", -1)
	regable = strings.Replace(regable, "*", ".*", -1)
	return regable
}

func regifyMysqlKeyString(name string) string {
	regable := strings.Replace(name, "..", ".", -1)
	regable = strings.Replace(regable, "{", "(", -1)
	regable = strings.Replace(regable, "}", ")", -1)
	regable = strings.Replace(regable, ",", "|", -1)
	regable = strings.Replace(regable, "*", ".*", -1)
	return regable
}

// in a path like moo.goo.* .. we find the initial "segment" that does not have a regex
func findFirstNonRegexSegment(name string) string {
	spl := strings.Split(name, ".")
	outStr := []string{}
	for _, s := range spl {
		if needRegex(s) {
			return strings.Join(outStr, ".")
		}
		outStr = append(outStr, s)
	}
	return strings.Join(outStr, ".")
}

func fullRegString(name string) string {
	regable := regifyKeyString(name)
	// we need to make sure we add the "^" + "$" to the end of things as well
	if !strings.HasPrefix(regable, "^") {
		regable = "^" + regable
	}
	if !strings.HasSuffix(regable, "$") {
		regable = regable + "$"
	}
	return regable
}

// convert the "graphite regex" into something golang understands (just the "."s really)
// need to replace things like "moo*" -> "moo.*" but careful not to do "..*"
// the "graphite" globs of {moo,goo} we can do with (moo|goo) so convert { -> (, , -> |, } -> )
func regifyKey(name string) (*regexp.Regexp, error) {
	regable := fullRegString(name)
	return regexp.Compile(regable)
}

// change {xxx,yyy} -> * as that's all the go lang glob can handle
// and so we turn it into t regex post
func toGlob(metric string) (string, []string) {

	outgs := []string{}
	gotFirst := false
	pGlob := ""
	outStr := ""
	regStr := ""
	for _, _c := range metric {
		c := string(_c)
		switch c {
		case "{":
			gotFirst = true
			regStr += "("
		case "}":
			if gotFirst && len(pGlob) > 0 {
				outgs = append(outgs, pGlob)
				regStr += ")" //end regex
				outStr += "*" //glob
				gotFirst = false
			}
		case ",":
			if gotFirst {
				regStr += "|" // glob , -> regex |
			} else {
				outStr += c
			}
		default:
			if !gotFirst {
				outStr += c
			} else {
				pGlob += c
			}
			regStr += c

		}
	}
	// make a proper regex
	regStr = strings.Replace(regStr, "*", ".*", -1)
	if !strings.HasSuffix(outStr, "*") {
		outStr += "*"
	}

	return regStr, outgs
}

// parse a tag query of the form key{name=val, name=val...}
func ParseOpenTSDBTags(query string) (key string, tags repr.SortingTags, err error) {
	// find the bits inside the {}

	inner := ""
	collecting := false
	keyCollecting := true
	for _, char := range query {
		switch char {
		case '{':
			collecting = true
			keyCollecting = false
		case '}':
			collecting = false
		default:
			if collecting {
				inner += string(char)
			}
			if keyCollecting {
				key += string(char)
			}
		}
	}

	if len(inner) == 0 || collecting {
		return key, tags, fmt.Errorf("Invalid Tag query `{name=val, name=val}`")
	}
	t_arr := strings.Split(inner, ",")
	for _, tg := range t_arr {
		t_split := strings.Split(strings.TrimSpace(tg), "=")
		if len(t_split) == 2 {
			tags = tags.Set(t_split[0], t_split[1])
		}
	}

	return key, tags, nil

}
