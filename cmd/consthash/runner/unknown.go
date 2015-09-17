/*
  the "i have no idea" runner
*/

package runner

import (
	"fmt"
)

type UnknownRunner struct{}

func (job *UnknownRunner) Name() (name string) { return "unknown" }

func (job UnknownRunner) ProcessLine(line string) (key string, orig_line string, err error) {
	return "", "", fmt.Errorf("Unknonwn line")
}
