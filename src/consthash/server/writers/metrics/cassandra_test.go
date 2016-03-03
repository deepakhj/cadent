package metrics

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

	reader := NewCassandraMetrics()
	reader.Config(t_config)

	rdata, err := reader.Render("consthash.zipperwork.local.[a-z]tatsd*", "-1h", "now")
	js, _ = json.Marshal(rdata)
	t.Logf("consthash.zipperwork.local.[a-z]tatsd*: %v", prettyprint(js))
	t.Logf("ERR: %v", err)

	rdata, err = reader.Render("consthash.zipperwork.local.graphite-statsd.lrucache.*", "-1h", "now")
	js, _ = json.Marshal(rdata)
	t.Logf("consthash.zipperwork.local.graphite-statsd.lrucache.*: %v", prettyprint(js))
	t.Logf("ERR: %v", err)

}
