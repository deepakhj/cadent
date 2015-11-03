/*
   Graphite data runner, <key> <value> <time> <thigns> ...
   we allow there to be more <things> after value, so this is really "graphite style"
   space based line entries with the key as the first item
*/

package runner

import (
	"fmt"
	"strings"
)

/****************** RUNNERS *********************/
const GRAPHITE_NAME = "graphite"

type GraphiteRunner struct {
	key_index int
}

func (job *GraphiteRunner) Name() (name string) { return GRAPHITE_NAME }

func NewGraphiteRunner(conf map[string]interface{}) (*GraphiteRunner, error) {

	//<key> <value> <time> <thigns>
	job := &GraphiteRunner{
		key_index: 0,
	}
	// allow for a config option to pick the proper thing in the line
	if idx, ok := conf["key_index"].(int); ok {
		job.key_index = idx
	}
	return job, nil
}

func (job *GraphiteRunner) ProcessLine(line string) (key string, orig_line string, err error) {
	//<key> <value> <time> <more> <more>
	//graphite_array := strings.Fields(line)
	graphite_array := strings.Split(line, " ")
	if len(graphite_array) > job.key_index {
		return graphite_array[job.key_index], line, nil
	}
	return "", "", fmt.Errorf("Invalid Graphite/Space line")

}
