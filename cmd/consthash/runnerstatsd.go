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
	Hasher    *ConstHasher
	param     string
	key_param string
	params    []string
}

func NewStatsdRunner(client Client, conf map[string]interface{}, param string) (*StatsdRunner, error) {

	//<key>:blaaa
	job := &StatsdRunner{
		client: client,
		Hasher: client.Hasher(),
		param:  param,
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

	servs, err := job.Hasher.GetN(job.GetKey(), job.Client().Server().Replicas)
	if err == nil {
		for _, useme := range servs {

			StatsdClient.Incr("success.valid-lines", 1)
			job.Client().Server().ValidLineCount.Up(1)
			sendOut := &SendOut{
				outserver: useme,
				param:     job.param,
				server:    job.Client().Server(),
				client:    job.Client(),
			}
			job.Client().Server().WorkerHold <- 1
			job.Client().WorkerQueue() <- sendOut
		}
		return fmt.Sprintf("yay statsd : %v : %v", servs, string(job.param))
	}
	StatsdClient.Incr("failed.invalid-hash-server", 1)
	job.Client().Server().UnsendableSendCount.Up(1)
	return fmt.Sprintf("ERROR ON statsd %s", err)
}
