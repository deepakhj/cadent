/*
   A Helper to basically list the Tags that are concidered "Id worthy"
   and those that are not
*/

package repr

var METRICS2_ID_TAGS []string = []string{
	"host",
	"http_method",
	"http_code",
	"device",
	"unit",
	"what",
	"type",
	"result",
	"stat",
	"bin_max",
	"direction",
	"mtype",
	"unit",
	"file",
	"line",
	"env",
}

var METRICS2_ID_TAGS_BYTES [][]byte = [][]byte{
	[]byte("host"),
	[]byte("http_method"),
	[]byte("http_code"),
	[]byte("device"),
	[]byte("unit"),
	[]byte("what"),
	[]byte("type"),
	[]byte("result"),
	[]byte("stat"),
	[]byte("bin_max"),
	[]byte("direction"),
	[]byte("mtype"),
	[]byte("unit"),
	[]byte("file"),
	[]byte("line"),
	[]byte("env"),
}

func IsMetric2Tag(name string) bool {
	for _, t := range METRICS2_ID_TAGS {
		if name == t {
			return true
		}
	}
	return false
}

func MergeMetric2Tags(newtags SortingTags, intag SortingTags, inmeta SortingTags) (SortingTags, SortingTags) {
	for _, ntag := range newtags {
		if IsMetric2Tag(ntag[0]) {
			intag.Set(ntag[0], ntag[1])
		} else {
			inmeta.Set(ntag[0], ntag[1])
		}
	}
	return intag, inmeta
}
