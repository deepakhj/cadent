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
	Interface bits for time series
*/

package series

// options for the series
type Options struct {
	NumValues          int64  `toml:"num_values"`
	HighTimeResolution bool   `toml:"high_time_resolution"`
	Handler            string `toml:"handler"`
}

func NewOptions(values int64, high_res bool) *Options {
	return &Options{
		NumValues:          values,
		HighTimeResolution: high_res,
		Handler:            "n/a",
	}
}

func NewDefaultOptions() *Options {
	return &Options{
		NumValues:          6,
		HighTimeResolution: false,
		Handler:            "n/a",
	}
}
