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

/** Gossip config elements **/

package config

import "cadent/server/gossip"

type GossipConfig struct {
	Enabled bool   `toml:"enabled" json:"enabled,omitempty"`
	Port    int    `toml:"port" json:"port,omitempty"` //
	Mode    string `toml:"mode" json:"mode,omitempty"` // local, lan, wan
	Name    string `toml:"name" json:"name,omitempty"` // name of this node, otherwise it will pick on (must be unique)
	Bind    string `toml:"bind" json:"bind,omitempty"` // bind ip
	Seed    string `toml:"seed" json:"seed,omitempty"` // a seed node
}

func (c *GossipConfig) Start() {
	// see if we can join up to the Gossip land
	if c.Enabled {
		_, err := gossip.Start(c.Mode, c.Port, c.Name, c.Bind)

		if c.Seed == "" {
			log.Noticef("Starting Gossip (master node) on port:%d seed: master", c.Port)
		} else {
			log.Noticef("Joining gossip on port:%d seed: %s", c.Port, c.Seed)
			err = gossip.CadentMembers.Join(c.Seed)

		}

		if err != nil {
			panic("Failed to join gossip: " + err.Error())
		}

	}
}
