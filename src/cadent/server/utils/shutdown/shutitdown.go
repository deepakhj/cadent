/*
Copyright 2016 Under Armour, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
var _SHUTDOWN_WAITGROUP sync.WaitGroup

func AddToShutdown() {
	_SHUTDOWN_WAITGROUP.Add(1)
}

func ReleaseFromShutdown() {
	_SHUTDOWN_WAITGROUP.Done()
}

func WaitOnShutdown() {
	_SHUTDOWN_WAITGROUP.Wait()
}
