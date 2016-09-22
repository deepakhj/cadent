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
   This is the "standalone" API server, which will interface to the main
   Data store as well as use a "seed" to grab any members in a cluster of writers

   metrics/indexer interfaces

   example config
   [api]
        base_path = "/graphite/"
        listen = "0.0.0.0:8084"
        seed = "http://localhost:8083/graphite"
        #and / OR (it will get the resolutions from the seed
        resolutions = [int, int, int]

            [api.metrics]
            driver = "whisper"
            dsn = "/data/graphite/whisper"

            # this is the read cache that will keep the latest goods in ram
            read_cache_max_items=102400
            read_cache_max_bytes_per_metric=8192

            [api.indexer]
            driver = "leveldb"
            dsn = "/data/graphite/idx"
*/

package api

import (
	"cadent/server/writers/indexer"
	"cadent/server/writers/metrics"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/memberlist"
	"gopkg.in/op/go-logging.v1"
	"net"
	"net/http"
	"net/url"
	"time"
)

const DEFAULT_HTTP_TIMEOUT = 10 * time.Second

type SoloApiLoop struct {
	Conf    ApiConfig
	Api     *ApiLoop
	Metrics metrics.Metrics
	Indexer indexer.Indexer

	Members []string

	shutdown chan bool
	log      *logging.Logger

	ReadCache *metrics.ReadCache

	started bool
}

func (re *SoloApiLoop) getUrl(u *url.URL, timeout time.Duration) (*http.Response, error) {

	transport := &http.Transport{
		Dial: (&net.Dialer{
			Timeout: timeout,
		}).Dial,
		TLSHandshakeTimeout: timeout,
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true}, // ignore "bad" certs
	}

	client := &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}

	return client.Get(u.String())
}

// from the member list, ask each member for it's API info
func (re *SoloApiLoop) ComposeMembers(membs []*memberlist.Node) error {
	/*for _, n := range membs {
		n.Addr
	}*/
	return nil
}

func (re *SoloApiLoop) GetSeedData(seed *url.URL) (*InfoData, error) {

	r, err := re.getUrl(seed, DEFAULT_HTTP_TIMEOUT)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	info := new(InfoData)
	err = json.NewDecoder(r.Body).Decode(info)
	if err != nil {
		return nil, err
	}

	err = re.ComposeMembers(info.Members)
	return info, err

}

func (re *SoloApiLoop) Config(conf SoloApiConfig, resolution float64) (err error) {
	// first need to grab things from any seeds (if any)

	if len(conf.Seed) == 0 || len(conf.Resolutions) == 0 {
		return fmt.Errorf("`seed` or `resolutions` is required")
	}
	rl := new(ApiLoop)

	if len(conf.Seed) > 0 {
		parsed, err := url.Parse(conf.Seed)
		if err != nil {
			return err
		}
		s_data, err := re.GetSeedData(parsed)
		if err != nil {
			return err
		}

		if len(s_data.Resolutions) == 0 {
			return fmt.Errorf("Unable to determine resolutions from seed %s", conf.Seed)
		}
		rl.Config(conf.GetApiConfig(), float64(s_data.Resolutions[0][0]))

		rl.SetResolutions(s_data.Resolutions)
		re.Api = rl
		return nil
	} else {
		rl.Config(conf.GetApiConfig(), float64(conf.Resolutions[0]))
		for_res := [][]int{}
		for _, r := range conf.Resolutions {
			for_res = append(for_res, []int{int(r), 0})
		}
		rl.SetResolutions(for_res)
	}
	return nil
}

func (re *SoloApiLoop) Stop() {
	re.Api.Stop()
}

func (re *SoloApiLoop) Start() {
	re.Api.Start()
}
