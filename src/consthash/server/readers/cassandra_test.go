package readers

// you best have some form of cassandra up

import (
	//. "github.com/smartystreets/goconvey/convey"
	"bytes"
	"encoding/json"
	"testing"
)

func TestCassandraReader(t *testing.T) {

	prettyprint := func(b []byte) string {
		var out bytes.Buffer
		json.Indent(&out, b, "", "  ")
		return string(out.Bytes())
	}

	//some tester strings
	t_config := make(map[string]interface{})
	t_config["dsn"] = "192.168.99.100"

	reader := NewCassandraReader()
	reader.Config(t_config)

	data, err := reader.Find("consthash.zipperwork.local")
	js, _ := json.Marshal(data)
	t.Logf("consthash.zipperwork.local: %v", prettyprint(js))
	t.Logf("ERR: %v", err)

	data, err = reader.Find("consthash.zipperwork.local.*")
	js, _ = json.Marshal(data)
	t.Logf("consthash.zipperwork.local.*: %v", prettyprint(js))
	t.Logf("ERR: %v", err)

	data, err = reader.Find("consthash.zipperwork.local.a[a-z]*")
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

	rdata, err := reader.Render("consthash.zipperwork.local.[a-z]tatsd*", "-1h", "now")
	js, _ = json.Marshal(rdata)
	t.Logf("consthash.zipperwork.local.[a-z]tatsd*: %v", prettyprint(js))
	t.Logf("ERR: %v", err)

	rdata, err = reader.Render("consthash.zipperwork.local.graphite-statsd.lrucache.*", "-1h", "now")
	js, _ = json.Marshal(rdata)
	t.Logf("consthash.zipperwork.local.graphite-statsd.lrucache.*: %v", prettyprint(js))
	t.Logf("ERR: %v", err)

}
