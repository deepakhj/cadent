/*
   a Regex runner that takes an arbitrary regex to yank it's key from
*/

package main

import (
	"fmt"
	"regexp"
)

type RegExRunner struct {
	client          Client
	Hashers         []*ConstHasher
	param           string
	key_regex       *regexp.Regexp
	key_regex_names []string
	key_param       string
	params          []string
}

func NewRegExRunner(client Client, conf map[string]interface{}, param string) (*RegExRunner, error) {

	//<key>:blaaa
	job := &RegExRunner{
		client:  client,
		Hashers: client.Hashers(),
		param:   param,
	}
	job.key_regex = conf["regexp"].(*regexp.Regexp)
	job.key_regex_names = conf["regexpNames"].([]string)

	matched := job.key_regex.FindAllStringSubmatch(param, -1)[0]
	for i, n := range matched {
		//fmt.Printf("%d. match='%s'\tname='%s'\n", i, n, n1[i])
		if job.key_regex_names[i] == "Key" {
			job.key_param = n
		}
	}

	if len(job.key_param) > 0 {
		job.param = param
		job.params = matched
		return job, nil
	}
	return nil, fmt.Errorf("Invalid RegEx line")
}

func (job RegExRunner) Client() Client {
	return job.client
}

func (job RegExRunner) GetKey() string {
	return job.key_param
}

func (job RegExRunner) run() string {

	//replicate the data across our Lists
	out_str := ""
	for idx, hasher := range job.Hashers {

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
			out_str += fmt.Sprintf("yay regex %s: %s", servs, string(job.param))
		} else {

			StatsdClient.Incr("failed.invalid-hash-server", 1)
			job.Client().Server().UnsendableSendCount.Up(1)
			out_str += fmt.Sprintf("ERROR ON regex %s", err)
		}
	}
	return out_str
}
