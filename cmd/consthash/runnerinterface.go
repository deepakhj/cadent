/*
An interface that is the "runner" for various Line Processors
*/

package main

type Runner interface {
	ProcessLine(line string) (key string, orig_line string, err error)
}
