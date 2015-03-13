/*
  the "i have no idea" runner
*/

package main

type UnknownRunner struct {
	client  Client
	Hashers []*ConstHasher
	param   string
	params  []string
}

func (job UnknownRunner) Client() Client {
	return job.client
}

func (job UnknownRunner) run() string {
	StatsdClient.Incr("failed.unknown-lines", 1)
	job.Client().Server().UnknownSendCount.Up(1)
	return "ACK no idea what message i'm supposed to parse"
}

func (job UnknownRunner) GetKey() string {
	return ""
}
