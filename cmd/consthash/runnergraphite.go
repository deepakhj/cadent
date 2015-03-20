/*
   Graphite data runner, <time> <key> <value>
*/

package main

import (
	"fmt"
	"strings"
	"time"
)

/****************** RUNNERS *********************/

type GraphiteRunner struct{}

func NewGraphiteRunner(conf map[string]interface{}) (*GraphiteRunner, error) {

	//<key>:blaaa
	job := &GraphiteRunner{}
	return job, nil
}

func (job *GraphiteRunner) ProcessLine(line string) (key string, orig_line string, err error) {
	defer StatsdNanoTimeFunc("factory.graphite.process-time-ns", time.Now())
	//<time> <key> <value>
	graphite_array := strings.Split(line, " ")
	if len(graphite_array) == 3 {
		return graphite_array[1], line, nil
	}
	return "", "", fmt.Errorf("Invalid Graphite line")

}
