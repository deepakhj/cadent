/*
   The server we listen for our data
*/

package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
)

type ConstHashRunner interface {
	run() string
}
type StatsdRunner struct {
	hasher *ConstHashHasher
	param  string
}
type GraphiteRunner struct {
	hasher *ConstHashHasher
	param  string
}
type UnknownRunner struct {
	hasher *ConstHashHasher
	param  string
}

func (job StatsdRunner) run() string {

	useme, err := job.hasher.Get(job.param)

	return fmt.Sprintf("yay statsd %s : %s : %s <key> <value>", job.param, useme, err)
	//return fmt.Sprintf("ERROR ON statsd %s", err)
}

func (job GraphiteRunner) run() string {
	useme, err := job.hasher.Get(job.param)
	if err != nil {
		return fmt.Sprintf("yay graphite %s: <time> <key> <value>", useme)
	}
	return fmt.Sprintf("ERROR ON graphite %s", err)

}

func (job UnknownRunner) run() string {
	return "ACK no idea what message i'm supposed to parse"
}

func constHashRunner(job ConstHashRunner, out chan string) {
	out <- job.run() + "\n"
}

func constHashFactory(hasher *ConstHashHasher, input string) ConstHashRunner {
	array := strings.Split(input, " ")
	if len(array) == 2 {
		return StatsdRunner{hasher: hasher, param: input}
	} else if len(array) == 3 {
		return GraphiteRunner{hasher: hasher, param: input}
	}

	return UnknownRunner{hasher: hasher, param: ""}
}

func handleRequest(conn net.Conn, out chan string, hasher *ConstHashHasher) {
	defer close(out)

	for {
		line, err := bufio.NewReader(conn).ReadBytes('\n')
		if err != nil {
			return
		}

		job := constHashFactory(hasher, strings.TrimRight(string(line), "\n"))
		go constHashRunner(job, out)
	}
}

func handleSend(conn net.Conn, in <-chan string) {
	defer conn.Close()

	for {
		message := <-in
		if len(message) > 0 {
			log.Print(message)
			io.Copy(conn, bytes.NewBufferString(message))
		}
	}
}

func createServer(cfg *ConstHashConfig) (connection net.Listener, err error) {
	log.Printf("Binding server to %s://%s", cfg.Protocal, cfg.Listen)
	return net.Listen(cfg.Protocal, cfg.Listen)
}

func startServer(cfg *ConstHashConfig, hasher *ConstHashHasher) {
	server, err := createServer(cfg)
	if err != nil {
		return
	}

	for {
		conn, err := server.Accept()
		if err != nil {
			return
		}

		channel := make(chan string)
		go handleRequest(conn, channel, hasher)
		go handleSend(conn, channel)
	}
}
