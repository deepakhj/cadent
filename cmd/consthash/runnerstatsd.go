/*
   statsd Runner and parser <key>:<value>|<type>
*/

package main

import (
	"fmt"
	"strings"
)

type StatsdRunner struct {
	client    Client
	Hashers   *[]*ConstHasher
	param     string
	key_param string
	params    []string
}

func NewStatsdRunner(client Client, conf map[string]interface{}, param string) (*StatsdRunner, error) {

	//<key>:blaaa
	job := &StatsdRunner{
		client:  client,
		Hashers: client.Hashers(),
		param:   param,
	}
	statd_array := strings.Split(param, ":")
	if len(statd_array) == 2 {
		job.param = param
		job.key_param = statd_array[0]
		job.params = statd_array
		return job, nil
	}
	return nil, fmt.Errorf("Invalid Statsd line")
}

func (job StatsdRunner) Client() Client {
	return job.client
}

func (job StatsdRunner) GetKey() string {
	//<key>:blaaa
	return job.key_param
}

func (job StatsdRunner) run() string {

	//replicate the data across our Lists
	out_str := ""
	for idx, hasher := range *job.Hashers {

		// may have replicas inside the pool too that we need to deal with
		servs, err := hasher.GetN(job.GetKey(), job.Client().Server().Replicas)
		if err == nil {
			for nidx, useme := range servs {
				// just log the valid lines "once" total ends stats are WorkerValidLineCount
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
			out_str += fmt.Sprintf("yay statsd %s: %s", servs, string(job.param))
		} else {

			StatsdClient.Incr("failed.invalid-hash-server", 1)
			job.Client().Server().UnsendableSendCount.Up(1)
			out_str += fmt.Sprintf("ERROR ON statsd %s", err)
		}
	}
	return out_str
}
