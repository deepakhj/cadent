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
   Simple gossip protocal to find the other Cadent's out there
*/

package gossip

import (
	"github.com/hashicorp/memberlist"
	logging "gopkg.in/op/go-logging.v1"
	"strings"
)

var log = logging.MustGetLogger("gossip")

//singleton
var CadentMembers *Members

func Get() *Members {
	return CadentMembers
}

type Members struct {
	List *memberlist.Memberlist
	Name string
	Mode string
	Port int
	Bind string
	Conf *memberlist.Config
}

func Start(mode string, port int, name string, bind string) (ms *Members, err error) {
	li := new(Members)
	li.Name = name
	li.Mode = mode
	li.Port = port
	li.Bind = bind
	li.Conf = li.Config()
	membs, err := memberlist.Create(li.Conf)
	if err != nil {
		return nil, err
	}
	li.List = membs

	if CadentMembers == nil {
		CadentMembers = li
	}
	return li, err

}

func (m *Members) Config() *memberlist.Config {
	var conf *memberlist.Config
	switch m.Mode {
	case "local":
		conf = memberlist.DefaultLocalConfig()
	case "lan":
		conf = memberlist.DefaultLANConfig()
	case "wan":
		conf = memberlist.DefaultWANConfig()
	default:
		panic("Gossip mode can be only local, lan, or wan")
	}

	if len(m.Bind) > 0 {
		conf.AdvertiseAddr = m.Bind
		conf.BindAddr = m.Bind
	}

	conf.AdvertisePort = m.Port
	conf.BindPort = m.Port

	if m.Name != "" {
		conf.Name = m.Name
	} else {
		m.Name = conf.Name
	}
	return conf
}

func (m *Members) Join(seed string) error {
	_, err := m.List.Join(strings.Split(seed, ","))
	if err != nil {
		panic("Failed to join cluster: " + err.Error())
	}
	return err
}

func (m *Members) Members(seed string) []*memberlist.Node {
	return m.List.Members()
}
