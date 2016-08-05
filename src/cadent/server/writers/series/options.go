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
