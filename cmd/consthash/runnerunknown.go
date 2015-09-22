/*
  the "i have no idea" runner
*/

package main

import (
	"fmt"
)

type UnknownRunner struct{}

func (job UnknownRunner) ProcessLine(line string) (key string, orig_line string, err error) {
	return "", "", fmt.Errorf("Unknonwn line")
}
