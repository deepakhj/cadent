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

package dispatch

/************************************************************************/
/**********  Standard Worker Dispatcher Queue ***************************/
/************************************************************************/
// insert job queue workers

type IJob interface {
	OnRetry() int
	IncRetry() int
	DoWork() error
}

type IWorker interface {
	Workpool() chan chan IJob
	Jobs() chan IJob
	Shutdown() chan bool

	Start() error
	Stop() error
}

type IDispatcher interface {
	Workpool() chan chan IJob
	JobsQueue() chan IJob
	Retries() int
	Shutdown()
	Run() error
}
