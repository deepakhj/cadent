/*
An interface that is the "runner" for various Line Processors
*/

package runner

type Runner interface {
	ProcessLine(line string) (key string, orig_line string, err error)
	Name() (name string)
}
