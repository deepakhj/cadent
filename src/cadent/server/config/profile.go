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

/** Profiling Health Server config elements **/

package config

import (
	"net/http"
	"runtime"
)

type ProfileConfig struct {
	Enabled      bool   `toml:"enabled" json:"enabled,omitempty"`
	ProfileBind  string `toml:"listen" json:"listen,omitempty"`
	ProfileRate  int    `toml:"rate" json:"rate,omitempty"`
	BlockProfile bool   `toml:"block_profile" json:"block_profile,omitempty"`
}

func (c *ProfileConfig) Start() {

	if c.Enabled {
		if c.ProfileRate > 0 {
			runtime.SetCPUProfileRate(c.ProfileRate)
			runtime.MemProfileRate = c.ProfileRate
		}
		if len(c.ProfileBind) > 0 {
			log.Notice("Starting Profiler on %s", c.ProfileBind)
			go http.ListenAndServe(c.ProfileBind, nil)
		}
	} else {
		// disable
		runtime.SetBlockProfileRate(0)
		runtime.SetCPUProfileRate(0)
		runtime.MemProfileRate = 0
	}

	// this can be "very expensive" so turnon lightly
	if c.BlockProfile {
		runtime.SetBlockProfileRate(1)
	}
}
