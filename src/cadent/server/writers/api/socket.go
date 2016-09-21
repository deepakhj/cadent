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
   WebSocket

   Here we allow a simple "attach" to a metric to watch for stats as they come in

    The mechanism attaches a channel listener to the ReadCache item, and tosses the json into
    socket
*/

package api

import (
	"cadent/server/repr"
	"cadent/server/stats"
	"cadent/server/writers/indexer"
	"cadent/server/writers/metrics"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"net/http"
)

var errWsNoData = []byte(fmt.Sprintf("%dNo data found", http.StatusNoContent))
var errWsTooManyMetrics = []byte(fmt.Sprintf("%dOnly one metric can be watched", http.StatusBadRequest))

type MetricsSocket struct {
	a       *ApiLoop
	Indexer indexer.Indexer
	Metrics metrics.Metrics

	upgrader websocket.Upgrader
}

func NewMetricsSocket(a *ApiLoop) *MetricsSocket {
	return &MetricsSocket{
		a:       a,
		Indexer: a.Indexer,
		Metrics: a.Metrics,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

func (m *MetricsSocket) AddHandlers(mux *mux.Router) {
	mux.HandleFunc("/metric", m.MetricSocket)
	//mux.HandleFunc("/websocket/all", m.AllMetricSocket)
}

func (re *MetricsSocket) MetricSocket(w http.ResponseWriter, r *http.Request) {

	stats.StatsdClientSlow.Incr("reader.http.websocket.connects", 1)
	c, err := re.upgrader.Upgrade(w, r, nil)
	// now that we've got it .. listen for entries
	if err != nil {
		re.a.log.Errorf("Error making socket: %v", err)
		return
	}
	defer func() {
		c.Close()
		re.a.log.Notice("Closing Web socket from %s", r.RequestURI)
	}()

	args, err := ParseMetricQuery(r)
	if err != nil {
		c.WriteMessage(websocket.CloseMessage, []byte(fmt.Sprintf("%d%v", http.StatusBadRequest, err)))
		re.a.OutError(w, fmt.Sprintf("%v", err), http.StatusBadRequest)
		return
	}

	data, err := re.Metrics.RawRender(args.Target, args.Start, args.End, args.Tags, args.Step)
	if err != nil {
		c.WriteMessage(websocket.CloseMessage, []byte(fmt.Sprintf("%d%v", http.StatusServiceUnavailable, err)))
		re.a.OutError(w, fmt.Sprintf("%v", err), http.StatusServiceUnavailable)
		return
	}
	if data == nil || len(data) == 0 {
		c.WriteMessage(websocket.CloseMessage, errWsNoData)
		re.a.OutError(w, "No data found", http.StatusNoContent)
		return
	}

	// we can onlyone metrics
	if len(data) > 1 {
		c.WriteMessage(websocket.CloseMessage, errWsTooManyMetrics)
		re.a.OutError(w, "Only one metric can be watched", http.StatusBadRequest)
		return
	}

	re.a.AddToCache(data)

	on_uid := data[0].Id
	re.a.log.Notice("Starting Web socket from %s", r.RequestURI)

	listen := re.a.ReadCache.ListenerChan()
	defer listen.Close()
	for {
		stat, more := <-listen.Ch
		if !more {
			return
		}
		if stat == nil {
			continue
		}
		ss := stat.(repr.StatRepr)
		if ss.Name.UniqueIdString() != on_uid {
			continue
		}
		err = c.WriteJSON(ss)
		stats.StatsdClient.Incr("reader.http.websocket.send", 1)
		if err != nil {
			stats.StatsdClient.Incr("reader.http.websocket.error", 1)
			re.a.log.Errorf("Error writing to socket: %v", err)
			break
		}
	}

	return
}
