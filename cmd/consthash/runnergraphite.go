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
	Hashers   *[]*ConstHasher
	key_param string
	param     string
	params    []string
}

func NewGraphiteRunner(client Client, conf map[string]interface{}, param string) (*GraphiteRunner, error) {

	// <time> <key> <value>
	job := &GraphiteRunner{
		client:  client,
		Hashers: client.Hashers(),
		param:   param,
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

	//replicate the data across our Lists
	out_str := ""
	for idx, hasher := range *job.Hashers {

		// may have replicas inside the pool too that we need to deal with
		servs, err := hasher.GetN(job.GetKey(), job.Client().Server().Replicas)
		if err == nil {
			for nidx, useme := range servs {

				if idx == 0 && nidx == 0 {
					job.Client().Server().ValidLineCount.Up(1)
					StatsdClient.Incr("success.valid-lines", 1)
				}
				StatsdClient.Incr("success.valid-lines-sent-to-workers", 1)
				job.Client().Server().WorkerValidLineCount.Up(1)

				sendOut := &SendOut{
					outserver: useme,
					server:    job.Client().Server(),
					param:     job.param,
					client:    job.Client(),
				}
				job.Client().Server().WorkerHold <- 1
				job.Client().WorkerQueue() <- sendOut
			}
			out_str += fmt.Sprintf("yay graphite %s: %s", servs, string(job.param))
		} else {

			StatsdClient.Incr("failed.invalid-hash-server", 1)
			job.Client().Server().UnsendableSendCount.Up(1)
			out_str += fmt.Sprintf("ERROR ON graphite %s", err)
		}
	}
	return out_str
}
