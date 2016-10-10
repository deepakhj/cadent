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
	"dc",
	"zone",
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
	[]byte("dc"),
	[]byte("zone"),
}

func IsMetric2Tag(name string) bool {
	for _, t := range METRICS2_ID_TAGS {
		if name == t {
			return true
		}
	}
	return false
}
func Metric2FromMap(intag map[string]string) (new_tag SortingTags, new_meta SortingTags) {

	for nkey, nval := range intag {
		if IsMetric2Tag(nkey) {
			new_tag = new_tag.Set(nkey, nval)
		} else {
			new_meta = new_meta.Set(nkey, nval)
		}
	}
	return new_tag, new_meta
}

func SplitIntoMetric2Tags(intag SortingTags, inmeta SortingTags) (SortingTags, SortingTags) {
	if inmeta.IsEmpty() && intag.IsEmpty() {
		return intag, inmeta
	}

	var new_meta SortingTags
	var new_tag SortingTags

	for _, ntag := range inmeta {
		if IsMetric2Tag(ntag.Name) {
			new_tag = new_tag.Set(ntag.Name, ntag.Value)
		} else {
			new_meta = new_meta.Set(ntag.Name, ntag.Value)
		}
	}
	for _, ntag := range intag {
		if IsMetric2Tag(ntag.Name) {
			new_tag = new_tag.Set(ntag.Name, ntag.Value)
		} else {
			new_meta = new_meta.Set(ntag.Name, ntag.Value)
		}
	}
	return new_tag, new_meta
}

func MergeMetric2Tags(newtags SortingTags, intag SortingTags, inmeta SortingTags) (SortingTags, SortingTags) {
	if newtags.IsEmpty() {
		return intag, inmeta
	}
	for _, ntag := range newtags {
		if IsMetric2Tag(ntag.Name) {
			intag = intag.Set(ntag.Name, ntag.Value)
		} else {
			inmeta = inmeta.Set(ntag.Name, ntag.Value)
		}
	}
	return intag, inmeta
}
