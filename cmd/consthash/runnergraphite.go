/*
   Graphite data runner, <time> <key> <value>
*/

package main

import (
	"fmt"
	"strings"
)

/****************** RUNNERS *********************/

type GraphiteRunner struct {
	client    Client
	Hasher    *ConstHasher
	key_param string
	param     string
	params    []string
}

func NewGraphiteRunner(client Client, conf map[string]interface{}, param string) (*GraphiteRunner, error) {

	//<key>:blaaa
	job := &GraphiteRunner{
		client: client,
		Hasher: client.Hasher(),
		param:  param,
	}
	graphite_array := strings.Split(param, " ")
	if len(graphite_array) == 3 {
		job.param = param
		job.key_param = graphite_array[1]
		job.params = graphite_array
		return job, nil
	}
	return nil, fmt.Errorf("Invalid Graphite line")
}

func (job GraphiteRunner) Client() Client {
	return job.client
}
func (job GraphiteRunner) GetKey() string {
	//<time> <key> <value>
	return job.key_param
}

func (job GraphiteRunner) run() string {
	// <time> <key> <value>
	useme, err := job.Hasher.Get(job.GetKey())
	if err == nil {
		StatsdClient.Incr("success.valid-lines", 1)
		job.Client().Server().ValidLineCount.Up(1)
		sendOut := &SendOut{
			outserver: useme,
			server:    job.Client().Server(),
			param:     job.param,
			client:    job.Client(),
		}
		job.Client().Server().WorkerHold <- 1

		job.Client().WorkerQueue() <- sendOut
		return fmt.Sprintf("yay graphite %s: %s", string(useme), string(job.param))
	}
	StatsdClient.Incr("failed.invalid-hash-server", 1)
	job.Client().Server().UnsendableSendCount.Up(1)
	return fmt.Sprintf("ERROR ON graphite %s", err)
}
