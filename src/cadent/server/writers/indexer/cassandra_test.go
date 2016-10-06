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

package indexer

// you best have some form of cassandra up

import (
	//. "github.com/smartystreets/goconvey/convey"
	"bytes"
	"cadent/server/repr"
	"encoding/json"
	"testing"
	"cadent/server/utils/options"
)

func TestCassandraReader(t *testing.T) {

	//skip it
	return
	prettyprint := func(b []byte) string {
		var out bytes.Buffer
		json.Indent(&out, b, "", "  ")
		return string(out.Bytes())
	}

	//some tester strings
	t_config := options.New()
	t_config.Set("dsn", "192.168.99.100")

	reader := NewCassandraIndexer()
	reader.Config(&t_config)
	tags := repr.SortingTags{}

	data, err := reader.Find("consthash.zipperwork.local", tags)
	js, _ := json.Marshal(data)
	t.Logf("consthash.zipperwork.local: %v", prettyprint(js))
	t.Logf("ERR: %v", err)

	data, err = reader.Find("consthash.zipperwork.local.*", tags)
	js, _ = json.Marshal(data)
	t.Logf("consthash.zipperwork.local.*: %v", prettyprint(js))
	t.Logf("ERR: %v", err)

	data, err = reader.Find("consthash.zipperwork.local.a[a-z]*", tags)
	js, _ = json.Marshal(data)
	t.Logf("consthash.zipperwork.local.a[a-z]*: %v", prettyprint(js))
	t.Logf("ERR: %v", err)

	edata, err := reader.Expand("consthash.zipperwork.local")
	js, _ = json.Marshal(edata)
	t.Logf("consthash.zipperwork.local: %v", prettyprint(js))
	t.Logf("ERR: %v", err)

	edata, err = reader.Expand("consthash.zipperwork.local.*")
	js, _ = json.Marshal(edata)
	t.Logf("consthash.zipperwork.local.*: %v", prettyprint(js))
	t.Logf("ERR: %v", err)

	edata, err = reader.Expand("consthash.zipperwork.local.[a-z]tatsd*")
	js, _ = json.Marshal(edata)
	t.Logf("consthash.zipperwork.local.[a-z]tatsd*: %v", prettyprint(js))
	t.Logf("ERR: %v", err)

}
