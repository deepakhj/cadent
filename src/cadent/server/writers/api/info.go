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
  Simple endpoint for getting "info" about this here server

  to be used for discovery of nodes around the land .. i.e. we give a server a "seed" server name
  it can then ping it and it can return back the list of other cadents out there

  this is NOT meant for a gossip like protocal, just for other services, there will
  eventually be an internal gossip protocal that wil drive the info from this endpoint

  i.e. one can use it if setting of an API frontend to discover all the cacher writer nodes out there
  (those writer/cacher nodes gossip to each other)

*/

package api

import (
	"cadent/server/gossip"
	"cadent/server/stats"
	"cadent/server/utils/shared"
	"cadent/server/writers/indexer"
	"cadent/server/writers/metrics"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/hashicorp/memberlist"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

type ApiInfoData struct {
	Host     string `json:"host"`
	BasePath string `json:"path"`
	Scheme   string `json:"scheme"`
	Port     string `json:"port"`
}

type InfoData struct {
	Time          int64              `json:"time"`
	MetricDriver  string             `json:"metric-driver"`
	IndexDriver   string             `json:"index-driver"`
	Resolutions   [][]int            `json:"resolutions"`
	Hostname      string             `json:"hostname"`
	Ip            []string           `json:"ip"`
	Members       []*memberlist.Node `json:"members"`
	CachedMetrics int                `json:"cached-metrics"`

	Api ApiInfoData `json:"api-server"`

	SharedData interface{} `json:"shared"`
}

type InfoAPI struct {
	a       *ApiLoop
	Indexer indexer.Indexer
	Metrics metrics.Metrics
}

func NewInfoAPI(a *ApiLoop) *InfoAPI {
	return &InfoAPI{
		a:       a,
		Indexer: a.Indexer,
		Metrics: a.Metrics,
	}
}

func (c *InfoAPI) AddHandlers(mux *mux.Router) {
	mux.HandleFunc("/info", c.GetInfo)
}

func (c *InfoAPI) GetInfo(w http.ResponseWriter, r *http.Request) {

	stats.StatsdClientSlow.Incr("reader.http.info.hit", 1)
	name, err := os.Hostname()
	if err != nil {
		name = fmt.Sprintf("{Could not get Hostname: %v}", err)
	}
	addrs, err := net.LookupHost(name)
	driver := c.Metrics.Driver()
	indexer := c.Indexer.Name()

	var res [][]int
	if c.Metrics != nil {
		res = c.Metrics.GetResolutions()
	}
	data := InfoData{
		Time:         time.Now().UnixNano(),
		MetricDriver: driver,
		IndexDriver:  indexer,
		Resolutions:  res,
		Hostname:     name,
		Ip:           addrs,
	}

	if c.Metrics != nil && c.Metrics.Cache() != nil {
		data.CachedMetrics = c.Metrics.Cache().Len()
	}

	if gossip.Get() != nil {
		data.Members = gossip.Get().Members()
		if len(data.Members) == 0 {
			data.Members = nil
		}
	} else {
		data.Members = nil
	}

	api_server := ApiInfoData{
		Host:     name,
		BasePath: c.a.Conf.BasePath,
		Scheme:   "http",
	}

	if len(c.a.Conf.TLSCertPath) > 0 && len(c.a.Conf.TLSKeyPath) > 0 {
		api_server.Scheme = "https"
	}
	spl := strings.Split(c.a.Conf.Listen, ":")
	api_server.Port = spl[len(spl)-1]
	data.Api = api_server

	data.SharedData = shared.GetAll()

	stats.StatsdClientSlow.Incr("reader.http.info.ok", 1)
	c.a.OutJson(w, data)

}
