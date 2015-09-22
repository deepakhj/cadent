/*
  the "i have no idea" runner
*/

package runner

import (
	"fmt"
)

const UNKNOWN_NAME = "unknown"

type UnknownRunner struct{}

func (job *UnknownRunner) Name() (name string) { return UNKNOWN_NAME }

func (job UnknownRunner) ProcessLine(line string) (key string, orig_line string, err error) {
	return "", "", fmt.Errorf("Unknonwn line")
}
