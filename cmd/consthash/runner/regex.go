/*
   a Regex runner that takes an arbitrary regex to yank it's key from
*/

package runner

import (
	"fmt"
	"regexp"
)

const REGEX_NAME = "regex"

type RegExRunner struct {
	key_regex       *regexp.Regexp
	key_regex_names []string
}

func (job *RegExRunner) Name() (name string) { return REGEX_NAME }

func NewRegExRunner(conf map[string]interface{}) (*RegExRunner, error) {

	//<key>:blaaa
	job := &RegExRunner{}
	job.key_regex = conf["regexp"].(*regexp.Regexp)
	job.key_regex_names = conf["regexpNames"].([]string)
	return job, nil
}

func (job RegExRunner) ProcessLine(line string) (key string, orig_line string, err error) {
	var key_param string

	matched := job.key_regex.FindAllStringSubmatch(line, -1)[0]
	for i, n := range matched {
		//fmt.Printf("%d. match='%s'\tname='%s'\n", i, n, n1[i])
		if job.key_regex_names[i] == "Key" {
			key_param = n
		}
	}
	if len(key_param) > 0 {
		return key_param, line, nil
	}
	return "", "", fmt.Errorf("Invalid Regex (cannot find key) line")

}
