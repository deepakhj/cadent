/*
   statsd Runner and parser <key>:<value>|<type>
*/

package main

import (
	"fmt"
	"strings"
	"time"
)

type StatsdRunner struct {
}

func NewStatsdRunner(conf map[string]interface{}) (*StatsdRunner, error) {

	//<key>:blaaa
	job := &StatsdRunner{}
	return job, nil
}

func (job StatsdRunner) ProcessLine(line string) (key string, orig_line string, err error) {
	defer StatsdNanoTimeFunc("factory.statsd.process-time-ns", time.Now())

	statd_array := strings.Split(line, ":")
	if len(statd_array) == 2 {
		return statd_array[0], line, nil
	}
	return "", "", fmt.Errorf("Invalid Statsd line")

}
