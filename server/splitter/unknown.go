/*
  the "i have no idea" runner
*/

package splitter

import (
	"fmt"
)

const UNKNOWN_NAME = "unknown"

type UnknownSplitter struct{}

func (job *UnknownSplitter) Name() (name string) { return UNKNOWN_NAME }

func (job UnknownSplitter) ProcessLine(line string) (key string, orig_line string, err error) {
	return "", "", fmt.Errorf("Unknonwn line")
}
