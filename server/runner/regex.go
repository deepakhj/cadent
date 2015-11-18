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
	key_index       int
}

func (job *RegExRunner) Name() (name string) { return REGEX_NAME }

func NewRegExRunner(conf map[string]interface{}) (*RegExRunner, error) {

	//<key>:blaaa
	job := &RegExRunner{}
	job.key_regex = conf["regexp"].(*regexp.Regexp)
	job.key_regex_names = conf["regexpNames"].([]string)
	// get the "Key" index for easy lookup
	for i, n := range job.key_regex_names {
		if n == "Key" {
			job.key_index = i
			break
		}
	}

	//job.key_regex_names := job.key_regex.SubexpNames()
	return job, nil
}

func (job RegExRunner) ProcessLine(line string) (key string, orig_line string, err error) {
	var key_param string

	matched := job.key_regex.FindAllStringSubmatch(line, -1)
	if matched == nil {
		return "", "", fmt.Errorf("Regex not matched")
	}
	if len(matched[0]) < (job.key_index + 1) {
		return "", "", fmt.Errorf("Named matches not found")
	}
	key_param = matched[0][job.key_index+1] // first string is always the original string

	if len(key_param) > 0 {
		return key_param, line, nil
	}
	return "", "", fmt.Errorf("Invalid Regex (cannot find key) line")

}
