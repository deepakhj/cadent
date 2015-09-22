/*
   statsd Runner and parser <key>:<value>|<type>
*/

package runner

import (
	"fmt"
	"strings"
)

const STATSD_NAME = "statsd"

type StatsdRunner struct {
}

func (job *StatsdRunner) Name() (name string) { return STATSD_NAME }

func NewStatsdRunner(conf map[string]interface{}) (*StatsdRunner, error) {

	//<key>:blaaa
	job := &StatsdRunner{}
	return job, nil
}

func (job StatsdRunner) ProcessLine(line string) (key string, orig_line string, err error) {

	statd_array := strings.Split(line, ":")
	if len(statd_array) >= 2 {
		return statd_array[0], line, nil
	}
	return "", "", fmt.Errorf("Invalid Statsd line: " + line)

}
