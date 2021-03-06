
# API/Readers

Readers are an attempt to imitate the Graphite API bits and include 3 main endpoints with a few more endpoints.

NOTE: _In the Works_ gRPC interface .. lots of protobuffing and testing still todo, but getting there

### TimeSeries

    /{root}/metrics -- get metrics in the format graphite needs
                        ?target=path.*.to.my.*.metric&from=time&to=time
    /{root}/rawrender -- get the actual metrics in the internal format
                        ?target=path.*.to.my.*.metric&from=time&to=time
    /{root}/cached/series -- get the BINARY blob of data only ONE metric allowed here
                        ?to=now&from=-10m&target=graphitetest.there.now.there
                          To enable base64
                        ...&base64=1
    /{root}/cache -- get the actual metrics stored in writeback cache
                        ?target=path.*.to.my.*.metric&from=time&to=time

### Path Indexer

    /{root}/find  -- find paths
                    ?query=path.*.to.my.*.metric
    /{root}/expand -- expand a tree
                    ?query=path.*.to.my.*.metric
    /{root}/list -- list all metric names limited to 2048 at a time
                    ?page=N


### Tag Indexer (in the works) (regexes for some backends may not be supported)

    /{root}/tag/find/byname -- find tags values by name (?name=host) this can be a typeahead form
                            ?name=ho.*
    /{root}/tag/find/bynamevalue -- find tags values by name and value `value`
                                    `value` can be of typeahead/regex form NOT the name
                                    ?name=host&value=moo.*
    /{root}/tag/uid/bytags -- get uniqueIdStrings no regexes allowed here.
                               You can omit the `metric_key` as well.
                             ?query=metric_key{name=val,name=val}
                             ?query={name=val,name=val}

### Graphite Mimics

    /{root}/render  -- gives back what graphite would give back in json format
                        ?target=path.*.to.my.*.metric&from=time&to=time
    /{root}/metrics/find  -- the basic find format graphite expects
                        ?query=path.*.to.my.*.metric

### Prometheus Mimics

    /prometheus/api/v1/query_range  -- get data in an expect Prometheus format
                        ?query=metric{tag="val", tag="val"}&start=XXX&end=XXX&step=X
	/prometheus/api/v1/label/__name__/values -- list all metric keys only 2048 at a time
	                    ?page=N
	/prometheus/api/v1/label/{name}/values -- tag value getter only 2048 at a time
	                    ?page=N


### Info

    /{root}/info  -- A big old json blob that is how this server the configured, and gossip members

### WebSockets (Reeeaaalllyy experimental)

    /ws/metric  -- Attach to a websocket, as stats get flushed, pop you get a new one
                    The metric you query must be "exact" (no search/regexes/finder things here)
                    ?query=path.to.metric
                    ?query={uid}



*NOTE both the Graphite and Prometheus lack the "function" (DSL) aspects so don't expect things like max(path.to.metric) to work*


For now, all return types are JSON, except the `/{root}/cached/series` which is binary/base64.

Items `/{root}/metrics` will be interpolated for have "nils" for data points that do not exist
(graphite expects a nice `{to - from}/step` span in the return vector).  `rawrender, cache*` endpoints will have just
the points that exists.

Upcoming Tag stuff

Indexing tags properly requires basically a trigram/inverted Lucene
like index.  Which we can impliment using ElasticSearch (or Solr).  However, most of the other DBs included here
(leveldb, mysql) do support some level of internal filtering, but mostly only in the "typeahead" sence
(i.e. find things like `ho.*`, then `hos.*` ..).  Cassandra is pretty bad here as in order to do that sort of typehead
filtering we acctually need to query and filter then entire tags space, which is not a good thing for big volumes.
So while we can use cassandra as the "tag -> UniqueId" map, we still need a way to find the tags we want first.

Mysql here is pretty good in that we can easily search for `select * from tags where name='ho%'`.  LevelDB/BoltDB
are also made for this very sort of prefix filtering as well, but are localized to just one machine (which might be ok
for your use case).


## Table of implemented apis and writers

Not everything can implement the APIs due to their nature. Below is a table of what drivers implement which endpoints

### Metrics

| Driver   | /rawrender + /metrics | /cache + /cached/series  | TagSupport |
|---|---|---|---|
| cassandra | Yes | Yes | No |
| cassandra-flat | Yes  | n/a| No |
| mysql | Yes  | Yes  |  Yes  |
| elasticsearch-flat | Yes  | No  |  Yes  |
| mysql-flat | Yes  | n/a | No |
| kafka | n/a  | Yes | n/a |
| kafka-flat | n/a  | n/a | n/a |
| levelDB | No  | No | No |
| file | n/a | n/a  | n/a |
| whisper| yes | n/a | n/a |


### Index

| Driver   |  /expand | /find  | TagSupport |
|---|---|---|---|---|---|
| cassandra | Yes | Yes | No |
| mysql | Yes  | Yes  |  Yes (alpha) |
| elasticsearch | Yes  | Yes  |  Yes  |
| kafka | n/a  | n/a | n/a |
| levelDB | Yes  | Yes | No |
| whisper | yes | yes | n/a |


`n/a` means it cannot/won't be implemented

`No` means it has not been implemented yet, but can

`TagSupport` is forth comming, but it will basicall add an extra Query param `tag=XXX` to things once the indexing has been hashed out

## Aggregation

Since graphite does not have the ability to tell the api what sort of aggregation we want/need from a given metric.  Cadent
attempts to infer what aggregation to use. Below is the mappings we use to infer, anything that does not match will get
the default of `mean`.  By "ends with" we mean the last verb in the metric name "moo.goo.endswith"

| MetricName  |  AggMethod |
|---|---|
| ends with: count(s?) |  sum |
| ends with: hit(s?) |  sum |
| ends with: ok |  sum |
| ends with: error(s?) |  sum |
| ends with: delete(s\|d?) |  sum |
| ends with: insert(s\|ed?) |  sum |
| ends with: update(s\|d?) |  sum |
| ends with: request(s\|ed?) |  sum |
| ends with: select(s\|ed?) |  sum |
| ends with: add(s\|ed)? |  sum |
| ends with: remove(s\|d?) |  sum |
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
| starts with: stats_count |  sum |
| starts with: stats.set |  sum |
| ends with: median |  median |
| ends with: middle |  median |
| ends with: median_\d+ |  median |
| DEFAULT | mean |


## Tag API

Since there can easily be some insanely bad queries (`name=* for instance`) All things are limited to 2048 items returned
(even this is alot), but think of what would happen if you tried to do a graphite query of `*.*.*.*.*`.

For tags, we will use the OpenTSDB format which is of the form `metric_key{name=val, name=val, ...}`

For the metrics 2.0 world, the `metric_key` is redendent and can be omitted and just use the tags.


## API Reader config

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
            # include these if you want a https endpoint
            # key="/path/to/server.key"
            # cert="/path/to/server.crt"

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
either "graphite-web" or "graphite-api". And you will need https://github.com/wyndhblb/pycandent

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
            tags: [{ name:"env", value: "prod"}, ...],
            meta_tags: [{ name:"env", value: "prod"}, ...],
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
    X-Cadentseries-Metatags:[{ name:"env", value: "prod"}, ...] | null
    X-Cadentseries-Points:11
    X-Cadentseries-Resolution:5
    X-Cadentseries-Start:1471386590000000000
    X-Cadentseries-Tags:[{ name:"env", value: "prod"}, ...] | null
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
        tags: [{ name:"env", value: "prod"}, ...],
        meta_tags: [{ name:"env", value: "prod"}, ...]
    },
    {
        text: "there",
        expandable: 0,
        leaf: 1,
        id: "graphitetest.there.now.now",
        path: "graphitetest.there.now.now",
        allowChildren: 0,
        uniqueid: "3w5dnlrj3clw3",
        tags: [{ name:"env", value: "prod"}, ...],
        meta_tags: [{ name:"env", value: "prod"}, ...]
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
