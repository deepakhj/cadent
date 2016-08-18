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

// a little util that is like "sync.Once" except we can reset it
// See http://golang.org/pkg/sync/#Once

package utils

import (
	"sync"
	"sync/atomic"
)

type Once struct {
	m    sync.Mutex
	done uint32
}

func (o *Once) Do(f func()) {
	if atomic.LoadUint32(&o.done) == 1 {
		return
	}

	o.m.Lock()
	defer o.m.Unlock()
	if o.done == 0 {
		defer atomic.StoreUint32(&o.done, 1)
		f()
	}
}

func (o *Once) Reset() {
	o.m.Lock()
	defer o.m.Unlock()
	atomic.StoreUint32(&o.done, 0)
}

// start/stop once in a group
type StartStop struct {
	start Once
	stop  Once
}

func (o *StartStop) Start(f func()) {
	o.start.Do(func() {
		f()
		o.stop.Reset()
	})
}

func (o *StartStop) Stop(f func()) {
	o.stop.Do(func() {
		f()
		o.start.Reset()
	})
}
