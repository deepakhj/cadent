# StatsD client (Golang)

## Introduction

Go Client library for [StatsD](https://github.com/etsy/statsd/). Contains a direct and a buffered client.
The buffered version will hold and aggregate values for the same key in memory before flushing them at the defined frequency.

This client library was inspired by the one embedded in the [quipo](https://github.com/quipo/statsd) project
extended to retain keys, recycle connections, and send buffers

## Installation
	
This is a bit tricky as it's a private repo
	
	cd my/code/path/
	git clone git@scm-main-01.dc.myfitnesspal.com:goutil/statsd.git
	
then in your import work

	import "./statsd"
	

## Supported event types

* Increment - Count occurrences per second/minute of a specific event
* Decrement - Count occurrences per second/minute of a specific event
* Timing - To track a duration event
* Gauge - Gauges are a constant data type. They are not subject to averaging, and they don’t change unless you change them. That is, once you set a gauge value, it will be a flat line on the graph until you change it again
* Absolute - Absolute-valued metric (not averaged/aggregated)
* Total - Continously increasing value, e.g. read operations since boot


## Sample usage

```go
package main

import (
    "time"

	"./statsd"
)

func main() {
	// init
	prefix := "myproject."
	statsdclient := statsd.NewStatsdClient("localhost:8125", prefix)
	statsdclient.CreateSocket()
	interval := time.Second * 2 // aggregate stats and flush every 2 seconds
	stats := statsd.NewStatsdBuffer(interval, statsdclient)
	defer stats.Close()

	// not buffered: send immediately
	statsdclient.Incr("mymetric", 4)

	// buffered: aggregate in memory before flushing
	stats.Incr("mymetric", 1)
	stats.Incr("mymetric", 3)
	stats.Incr("mymetric", 1)
	stats.Incr("mymetric", 1)
}
```

The string "%HOST%" in the metric name will automatically be replaced with the hostname of the server the event is sent from.


## Author

bobalanton@myfitnesspal.com


## Copyright

See [LICENSE](LICENSE) document
