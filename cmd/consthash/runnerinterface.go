/*
An interface that is the "runner" for various hashers
*/

package main

type Runner interface {
	run() string    //"do" the work and push it to the work queue
	Client() Client // the tcp/udp client connections
	GetKey() string // the "key" that we need to hash
}
