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

package cadent

import (
	"statsd"

	"cadent/server/stats"
	"fmt"
	. "github.com/smartystreets/goconvey/convey"
	"io/ioutil"
	"net"
	"net/url"
	"os"
	"testing"
)

func TesterTCPMockListener(inurl string) net.Listener {
	i_url, _ := url.Parse(inurl)
	net, err := net.Listen(i_url.Scheme, i_url.Host)
	if err != nil {
		panic(err)
	}
	return net
}

func TesterUDPMockListener(inurl string) *net.UDPConn {
	i_url, _ := url.Parse(inurl)
	udp_addr, err := net.ResolveUDPAddr(i_url.Scheme, i_url.Host)

	net, err := net.ListenUDP(i_url.Scheme, udp_addr)
	if err != nil {
		panic(err)
	}
	return net
}

var confs ConfigServers
var defc *Config
var useconfigs []*Config
var statsclient statsd.Statsd
var servers []*Server

var stats_listen *net.UDPConn

func TestConsthashConfigDecode(t *testing.T) {

	confstr := `
[default]
pid_file="/tmp/consthash.pid"
num_procs=0
max_pool_connections=10
sending_method="pool"
hasher_algo="md5"
hasher_elter="graphite"
hasher_vnodes=100
num_dupe_replicas=1
cache_items=50000
heartbeat_time_delay=5
heartbeat_time_timeout=1
failed_heartbeat_count=3
server_down_policy="remove_node"
cpu_profile=true
statsd_server="127.0.0.1:5000"
statsd_prefix="consthash"
statsd_interval=1  # send to statd every second (buffered)
workers=5
internal_health_server_listen="0.0.0.0:6061"
internal_health_server_points=500
internal_health_server_path="html"

[statsd-example]
listen="udp://0.0.0.0:32121"
msg_type="statsd"
hasher_algo="nodejs-hashring"
hasher_elter="statsd"
hasher_vnodes=40

  [[statsd-example.servers]]
  servers=["udp://192.168.59.103:8126", "udp://192.168.59.103:8136", "udp://192.168.59.103:8146"]
  check_servers=["tcp://192.168.59.103:8127", "tcp://192.168.59.103:8137", "tcp://192.168.59.103:8147"]

  [[statsd-example.servers]]
  servers=["udp://192.168.59.104:8126", "udp://192.168.59.104:8136", "udp://192.168.59.104:8146"]
  check_servers=["tcp://192.168.59.104:8127" ,"tcp://192.168.59.104:8137", "tcp://192.168.59.104:8147"]

[graphite-example]
listen="tcp://0.0.0.0:33232"
msg_type="graphite"

  [[graphite-example.servers]]
  servers=["tcp://192.168.59.103:2003", "tcp://192.168.59.103:2004", "tcp://192.168.59.103:2005"]

[graphite-backend]
listen="backend_only"
msg_type="graphite"

  [[graphite-backend.servers]]
  servers=["tcp://192.168.59.105:2003", "tcp://192.168.59.105:2004", "tcp://192.168.59.105:2005"]

`

	// for comparixon
	var serverlisters = make(map[string][][]string)

	serverlisters["statsd-example"] = [][]string{
		{"udp://192.168.59.103:8126", "udp://192.168.59.103:8136", "udp://192.168.59.103:8146"},
		{"udp://192.168.59.104:8126", "udp://192.168.59.104:8136", "udp://192.168.59.104:8146"},
	}
	serverlisters["graphite-example"] = [][]string{
		{"tcp://192.168.59.103:2003", "tcp://192.168.59.103:2004", "tcp://192.168.59.103:2005"},
	}
	serverlisters["graphite-backend"] = [][]string{
		{"tcp://192.168.59.105:2003", "tcp://192.168.59.105:2004", "tcp://192.168.59.105:2005"},
	}

	confs, _ = ParseConfigString(confstr)
	defc, _ = confs.DefaultConfig()
	useconfigs = confs.ServableConfigs()

	// nexted loops have issues for Convey
	Convey("Given a config string", t, func() {

		Convey("We should have 1 default server section", func() {
			So(defc, ShouldNotEqual, nil)
		})
		Convey("Default section sending method should be `pool`", func() {
			So(defc.SendingConnectionMethod, ShouldEqual, "pool")
		})
		Convey("We should have 3 main server section", func() {
			So(len(useconfigs), ShouldEqual, len(serverlisters))

			for _, serv := range useconfigs {
				if serv.Name == "statsd-example" {
					Convey(fmt.Sprintf("statsd should have non-default hasher confs `%s`", serv.Name), func() {
						So(serv.HashAlgo, ShouldEqual, "nodejs-hashring")
						So(serv.HashElter, ShouldEqual, "statsd")
						So(serv.HashVNodes, ShouldEqual, 40)
					})
				}
			}
		})
	})

	SetUpStatsdClient(defc)
	statsclient = stats.StatsdClient
	stats_listen = TesterUDPMockListener("udp://" + defc.StatsdServer)
	defer stats_listen.Close()

	Convey("We should be able to set up a statsd client", t, func() {
		So(statsclient.String(), ShouldEqual, defc.StatsdServer)
	})

	Convey("We should be able to create a statsd socket", t, func() {
		So(statsclient.CreateSocket(), ShouldEqual, nil)
		statsclient.Incr("moo.goo.org", 1)
	})

	// there is trouble with too much nesting in convey
	for _, cfg := range useconfigs {

		var hashers []*ConstHasher

		for _, serverlist := range cfg.ServerLists {
			hasher, err := CreateConstHasherFromConfig(cfg, serverlist)

			if err != nil {
				panic(err)
			}
			go hasher.ServerPool.StartChecks()
			hashers = append(hashers, hasher)
		}
		server, err := CreateServer(cfg, hashers)
		if err != nil {
			panic(err)
		}
		servers = append(servers, server)
		go server.StartServer()
	}

	Convey("We should be able start up and run the hashing servers", t, func() {

		Convey("Should have 3 server", func() {
			So(len(servers), ShouldEqual, 3)
		})

		Convey("Logging test out for coverage", func() {
			confs.DebugConfig()
		})
	})

	Convey("Given a config file", t, func() {
		fname := "/tmp/__cfg_tonfig.toml"
		ioutil.WriteFile(fname, []byte(confstr), 0644)
		defer os.Remove(fname)

		confs, _ := ParseConfigFile(fname)
		_, err := confs.DefaultConfig()
		Convey("We should have 1 default server section", func() {
			So(err, ShouldEqual, nil)
		})
		servers := confs.ServableConfigs()
		Convey("We should have 2 main server section", func() {
			So(len(servers), ShouldEqual, 3)
		})

	})
}
