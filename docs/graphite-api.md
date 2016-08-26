
### Graphite like API/Readers

Readers are an attempt to imitate the Graphite API bits and include 3 main endpoints with a few more endpoints


    /{root}/find  -- find paths ( ?query=path.*.to.my.*.metric )
    /{root}/expand -- expand a tree ( ?query=path.*.to.my.*.metric )
    /{root}/metrics -- get metrics in the format graphite needs ( ?target=path.*.to.my.*.metric&from=time&to=time )
    /{root}/rawrender -- get the actuall metrics in the internal format ( ?target=path.*.to.my.*.metric&from=time&to=time )
    /{root}/cached/series -- get the BINARY blob of data only ONE metric allowed here ( ?to=now&from=-10m&target=graphitetest.there.now.there )
    /{root}/cache -- get the actuall metrics stored in writeback cache ( ?target=path.*.to.my.*.metric&from=time&to=time )

For now, all return types are JSON.

Unlike the Whisper file format which keeps "nils" for no data (i.e. a round robin DB with a fixed step size and known counts),
the mature of the metrics in our various backends write points at what ever the flush time is, and if there is nothing to write
does not write "nils" so the `/metrics` endpoint has to return an interpolated set of data to attempt to match what graphite expects
(this is more a warning for those that may notice some time shifting in some data and "data" holes)

This may mean that you will see some random interspersed `nils` in the data on small time ranges.  There are a variety of reasons for this
1) flush times are not "exact" go's in the concurency world, not everything is run exactly when we want it do so over time some "drift" will occur.
2) Since we are both "flushing to disk" and "flushing from buffers" from many buffers at different times, sometimes they just don't line up

*NOTE*  Currently only Cassandra, MySQL and Whisper "render apis" are valid. File and Kafka writers can have no render apis.

*NOTE* the `/cached/series` endpoint only makes since for TimeSeries based writers, as a result only Cassandra, MySQL, kafka series writers impliment this

#### Table of implemented apis and writers

Not everything can implement the APIs due to their nature. Below is a table of what drivers implement which endpoints

##### Metrics

| Driver   | /rawrender + /metrics | /cache + /cached/series  | TagSupport |
|---|---|---|---|
| cassandra | Yes | Yes | No |
| cassandra-flat | Yes  | n/a| No |
| mysql | Yes  | Yes  |  No |
| mysql-flat | Yes  | n/a | No |
| kafka | n/a  | Yes | n/a |
| kafka-flat | n/a  | n/a | n/a |
| levelDB | No  | No | No |
| file | n/a | n/a  | n/a |
| whisper| yes | n/a | n/a |


##### Index

| Driver   |  /expand | /find  | TagSupport |
|---|---|---|---|---|---|
| cassandra | Yes | Yes | No |
| mysql | Yes  | Yes  |  No |
| kafka | n/a  | n/a | n/a |
| levelDB | Yes  | Yes | No |
| whisper | yes | yes | n/a |


`n/a` means it cannot/won't be implemeneted

`No` means it has not been implemended yet

`TagSupport` is forth comming, but it will basicall add an extra Query param `tag=XXX` to things once the indexing has been hashed out

#### Aggregation

Since graphite does not have the ability to tell the api what sort of aggregation we want/need from a given metric.  Cadent
attempts to infer what aggregation to use. Below is the mappings we use to infer, anything that does not match will get
the default of `mean`.

| MetricName  |  AggMethod |
|---|---|
| ends with: count |  sum |
| ends with: error(s?) |  sum |
| ends with: delete(s|d?) |  sum |
| ends with: insert(s|ed?) |  sum |
| ends with: update(s|d?) |  sum |
| ends with: request(s|ed?) |  sum |
| ends with: select(s|ed?) |  sum |
| ends with: add(s|ed)? |  sum |
| ends with: remove(s?) |  sum |
| ends with: remove(s|d?) |  sum |
| ends with: consume(d?) |  sum |
| ends with: sum |  sum |
| ends with: max |  max |
| ends with: max_\d+ |  max |
| ends with: upper |  max |
| ends with: upper_\d+ |  max |
| ends with: min |  min |
| ends with: lower |  min |
| ends with: min_\d+ |  min |
| ends with: lower_\d+ |  min |
| ends with: gauge |  last |
| starts with: stats.gauge |  last |
| ends with: median |  median |
| ends with: middle |  median |
| starts with: median_\d+ |  median |
| DEFAULT | mean |



#### API Reader config

    [statsd-regex-map]
    listen_server="statsd-proxy" # which listener to sit in front of  (must be in the main config)
    default_backend="statsd-proxy"  # failing a match go here

        [statsd-regex-map.accumulator]
        backend = "BLACKHOLE"  # we are just writing to cassandra
        input_format = "statsd"
        output_format = "graphite"
        #keep_keys = true  #  will constantly emit "0" for metrics that have not arrived


        # options for statsd input formats (for outputs)
        options = [
            ["legacyNamespace", "true"],
            ["prefixGauge", "g"],
            ["prefixTimer", "t"],
            ["prefixCounter", "c"],
            ["globalPrefix", "ss"],
            ["globalSuffix", "test"],
            ["percentThreshold", "0.75,0.90,0.95,0.99"]
        ]

        # aggregate bin counts
        times = ["5s:168h", "1m:720h"]

        # writer of indexes and metrics (happen to be the same data source)
        [statsd-regex-map.accumulator.writer.metrics]
            driver = "cassandra"
            dsn = "192.168.99.100"
            [statsd-regex-map.accumulator.writer.metrics.options]
                user="cassandra"
                pass="cassandra"
        [statsd-regex-map.accumulator.writer.indexer]
            driver = "cassandra"
            dsn = "192.168.99.100"
            [statsd-regex-map.accumulator.writer.indexer.options]
                 user="cassandra"
                 pass="cassandra"

        # API options (yes they are the same as above, but there's nothing saying it has to be)
        [statsd-regex-map.accumulator.api]
            base_path = "/graphite/"
            listen = "0.0.0.0:8083"

            [statsd-regex-map.accumulator.api.metrics]
                driver = "cassandra"
                dsn = "192.168.99.100"
                [statsd-regex-map.accumulator.api.metrics.options]
                   user="cassandra"
                   pass="cassandra"
            [statsd-regex-map.accumulator.api.indexer]
                driver = "cassandra"
                dsn = "192.168.99.100"
                [statsd-regex-map.accumulator.api.indexer.options]
                    user="cassandra"
                    pass="cassandra"


This will fire up a http server listening on port 8083 for those 3 endpoints above.  In order to get graphite to "understand" this endpoint you can use
either "graphite-web" or "graphite-api". And you will need https://gitlab.mfpaws.com/Metrics/pycandent

For graphite-web you'll need to add these in the `settings.py` (based on the settings above)

    STORAGE_FINDERS = (
       'cadent.CadentFinder',
    )
    CADENT_TIMEZONE="America/Los_Angeles"
    CADENT_URLS = (
        'http://localhost:8083/graphite',
    )

For graphite-api add this to the yaml conf

    cadent:
        urls:
            - http://localhost:8083/graphite
    finders:
        - cadent.CadentFinder


## Outputs

Just some example output from the render apis

### /cache + /rawrender

    [
        {
            metric: "graphitetest.there.now.there",
            id: "3w5dnlrj3clw3",
            tags: [["env", "prod"], ...],
            meta_tags: [["foo", "baz"], ...],
            data_from: 1471359695,
            data_end: 1471359810,
            from: 1471359220,
            to: 1471359820,
            step: 5,
            aggfunc: 0,
            data: [
                {
                    time: 1471359695,
                    sum: 5152590,
                    min: 75544,
                    max: 84989,
                    last: 82659,
                    count: 64
                }, ...
            ]
        }, ...
    ]

### /metrics

    {
        data_from: 1471363625,
        data_end: 1471364215,
        from: 1471363625,
        to: 1471364220,
        step: 5,
        series: {
            graphitetest.there.now.there: [
                [
                    80248.617188,
                    1471363625
                ],
                [
                    80367.816514,
                    1471363630
                ],
                ...
            ],
            graphitetest.there.now.now: [ ... ]
        }
    }

### /cached/series

Note this one returns BINARY data in the body, but w/ some nice headers.  Multiple targets are not allowed here
as there is no "multi series" binary format.  You can look up things by the UniqueId as well `target=3w5dnlrj3clw3`.
The `from` and `to` are just used to pick the proper resolution, the series you get back will be whatever is in the cache
start and ends are in the Headers.

You can also request a base64 encoded version by including `&base64=1` in the GET.


    X-Cadentseries-Encoding:gorilla
    X-Cadentseries-End:1471386630000000000
    X-Cadentseries-Key:graphitetest.there.now.there
    X-Cadentseries-Metatags:[["env", "prod"], ...] | null
    X-Cadentseries-Points:11
    X-Cadentseries-Resolution:5
    X-Cadentseries-Start:1471386590000000000
    X-Cadentseries-Tags:[["moo", "goo"], ...] | null
    X-Cadentseries-Ttl:3600
    X-Cadentseries-Uniqueid:3w5dnlrj3clw3


    ....The the acctual binary data blob (in the raw form)....


### /find

    [
    {
        text: "there",
        expandable: 0,
        leaf: 1,
        id: "graphitetest.there.now.there",
        path: "graphitetest.there.now.there",
        allowChildren: 0,
        uniqueid: "3w5dnlrj3clw3",
        tags: [["env", "prod"], ...],
        meta_tags: [["foo", "baz"], ...]
    },
    {
        text: "there",
        expandable: 0,
        leaf: 1,
        id: "graphitetest.there.now.now",
        path: "graphitetest.there.now.now",
        allowChildren: 0,
        uniqueid: "3w5dnlrj3clw3",
        tags: [["env", "prod"], ...],
        meta_tags: [["foo", "baz"], ...]
    },
    ...
    ]

### /expand

    {
    results: [
        "graphitetest.there.now.badline",
        "graphitetest.there.now.cow",
        "graphitetest.there.now.here",
        "graphitetest.there.now.house",
        "graphitetest.there.now.now",
        "graphitetest.there.now.test",
        "graphitetest.there.now.there"
    ]
    }
