/*
	Utils for basic the key inputs for converting glob patters to something
	golang understands
*/

package indexer

import (
	"regexp"
	"strings"
)

// need regex?
func needRegex(metric string) bool {
	return strings.IndexAny(metric, "*?[{") >= 0
}

// convert the "graphite regex" into something golang understands (just the "."s really)
// need to replace things like "moo*" -> "moo.*" but careful not to do "..*"
// the "graphite" globs of {moo,goo} we can do with (moo|goo) so convert { -> (, , -> |, } -> )
func regifyKey(metric string) (*regexp.Regexp, error) {
	regable := strings.Replace(metric, "..", ".", -1)
	regable = strings.Replace(regable, "{", "(", -1)
	regable = strings.Replace(regable, "}", ")", -1)
	regable = strings.Replace(regable, ",", "|", -1)
	regable = strings.Replace(regable, ".", "\\.", -1)
	regable = strings.Replace(regable, "*", ".*", -1)
	return regexp.Compile(regable)
}

// change {xxx,yyy} -> * as that's all the go lang glob can handle
// and so we turn it into t regex post
func toGlob(metric string) (string, []string) {

	outgs := []string{}
	got_first := false
	p_glob := ""
	out_str := ""
	reg_str := ""
	for _, _c := range metric {
		c := string(_c)
		switch c {
		case "{":
			got_first = true
			reg_str += "("
		case "}":
			if got_first && len(p_glob) > 0 {
				outgs = append(outgs, p_glob)
				reg_str += ")" //end regex
				out_str += "*" //glob
				got_first = false
			}
		case ",":
			if got_first {
				reg_str += "|" // glob , -> regex |
			} else {
				out_str += c
			}
		default:
			if !got_first {
				out_str += c
			} else {
				p_glob += c
			}
			reg_str += c

		}
	}
	// make a proper regex
	reg_str = strings.Replace(reg_str, "*", ".*", -1)
	if !strings.HasSuffix(out_str, "*") {
		out_str += "*"
	}

	return reg_str, outgs
}
