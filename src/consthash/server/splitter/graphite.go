/*
   Graphite data runner, <key> <value> <time> <thigns> ...
   we allow there to be more <things> after value, so this is really "graphite style"
   space based line entries with the key as the first item
*/

package splitter

import (
	"fmt"
	"strings"
)

/****************** RUNNERS *********************/
const GRAPHITE_NAME = "graphite"

type GraphiteSplitter struct {
	key_index int
}

func (job *GraphiteSplitter) Name() (name string) { return GRAPHITE_NAME }

func NewGraphiteSplitter(conf map[string]interface{}) (*GraphiteSplitter, error) {

	//<key> <value> <time> <thigns>
	job := &GraphiteSplitter{
		key_index: 0,
	}
	// allow for a config option to pick the proper thing in the line
	if idx, ok := conf["key_index"].(int); ok {
		job.key_index = idx
	}
	return job, nil
}

func (job *GraphiteSplitter) ProcessLine(line string) (key string, orig_line string, err error) {
	//<key> <value> <time> <more> <more>
	//graphite_array := strings.Fields(line)
	graphite_array := strings.Split(line, " ")
	if len(graphite_array) > job.key_index {
		return graphite_array[job.key_index], line, nil
	}
	return "", "", fmt.Errorf("Invalid Graphite/Space line")

}
