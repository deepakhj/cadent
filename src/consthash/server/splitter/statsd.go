/*
   statsd Runner and parser <key>:<value>|<type>
*/

package splitter

import (
	"fmt"
	"strings"
)

const STATSD_NAME = "statsd"

type StatsdSplitter struct {
}

func (job *StatsdSplitter) Name() (name string) { return STATSD_NAME }

func NewStatsdSplitter(conf map[string]interface{}) (*StatsdSplitter, error) {

	//<key>:blaaa
	job := &StatsdSplitter{}
	return job, nil
}

func (job *StatsdSplitter) ProcessLine(line string) (key string, orig_line string, err error) {

	statd_array := strings.Split(line, ":")
	if len(statd_array) >= 2 {
		return statd_array[0], line, nil
	}
	return "", "", fmt.Errorf("Invalid Statsd line: " + line)

}
