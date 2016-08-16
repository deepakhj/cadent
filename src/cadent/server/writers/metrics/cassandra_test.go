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

package metrics

// you best have some form of cassandra up

import (
	//. "github.com/smartystreets/goconvey/convey"
	"bytes"
	"encoding/json"
	"testing"
)

func TestCassandraReader(t *testing.T) {
	// skip it
	return
	prettyprint := func(b []byte) string {
		var out bytes.Buffer
		json.Indent(&out, b, "", "  ")
		return string(out.Bytes())
	}

	//some tester strings
	t_config := make(map[string]interface{})
	t_config["dsn"] = "192.168.99.100"

	reader := NewCassandraMetrics()
	reader.Config(t_config)

	rdata, err := reader.Render("consthash.zipperwork.local.[a-z]tatsd*", "-1h", "now")
	js, _ := json.Marshal(rdata)
	t.Logf("consthash.zipperwork.local.[a-z]tatsd*: %v", prettyprint(js))
	t.Logf("ERR: %v", err)

	rdata, err = reader.Render("consthash.zipperwork.local.graphite-statsd.lrucache.*", "-1h", "now")
	js, _ = json.Marshal(rdata)
	t.Logf("consthash.zipperwork.local.graphite-statsd.lrucache.*: %v", prettyprint(js))
	t.Logf("ERR: %v", err)

}
