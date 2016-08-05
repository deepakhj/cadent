/*
 This is a helper WaitGroup singleton that helps with doing shutdowns in a nice fashion

 basically any "Stop()" or "Shutdown()" command should add it self to the waitgroup

  the "caller" of the shutdown should then mark it as done.

  the "root" caller of the shutdown (usually a SIGINT signal) then should wait for everything to finish
  and finally "exit(0)"

*/
package shutdown

import "sync"

//signelton
var _SHUTDOWN_WAITGROUP *sync.WaitGroup

func init() {
	_SHUTDOWN_WAITGROUP = new(sync.WaitGroup)
}

func AddToShutdown() {
	_SHUTDOWN_WAITGROUP.Add(1)
}

func ReleaseFromShutdown() {
	_SHUTDOWN_WAITGROUP.Done()
}

func WaitOnShutdown() {
	_SHUTDOWN_WAITGROUP.Wait()
}
