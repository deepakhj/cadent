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
   A simple "shared" place to put various data related to just about anythin we want
*/

package shared

import (
	"sync"
)

//singleton
type GlobalData map[string]interface{}

var data GlobalData
var mu *sync.RWMutex

func init() {
	data = make(GlobalData, 0)
	mu = new(sync.RWMutex)
}

func Get(name string) interface{} {
	mu.RLock()
	defer mu.RUnlock()
	return data[name]
}

func Set(name string, indata interface{}) {
	mu.Lock()
	defer mu.Unlock()
	data[name] = indata
}

func GetAll() GlobalData {
	return data
}
