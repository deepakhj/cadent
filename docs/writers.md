
### Writers

Accumulators can "write" to something other then a tcp/udp/http/socket, to say things like a FILE, MySQL DB or cassandra.
(since this is Golang all writer types need to have their own driver embed in).  If you only want accumulators to write out to
these things, you can specify the `backend` to `BLACKHOLE` which will NOT try to reinsert the line back into the pipeline
and the line "ends" with the Accumulator stage.


    InLine(port 8126) -> Splitter -> [Accumulator] -> WriterBackend

Writers should hold more then just the "aggregated data point" but a few useful things like

    Min, Max, Sum, First, Last and Count

because who's to say what you really want from aggregated values.
`Count` is just actually how many data points arrived in the aggregation window (for those wanting `Mean` (Sum / Count))

Some example Configs for the current "4" writer backends

Writers themselves are split into 2 sections "Indexers" and "Metrics"

Indexers: take a metrics "name" which has the form

    StatName{
        Key string
        Tags  [][]string
        Resolution uint32
        TTL uint32
    }

And will "index it" somehow.  Currently the "tags" are not yet used in the indexers .. but the future ....

The StatName has a "UniqueID" which is basically a MMH3 hash of the following

    FNV64a(Key + ":" + sortByName(Tags))

Metrics: The base "unit" of a metric is this

    StatMetric{
        Time int64
        Min float64
        Max float64
        First float64
        Last float64
        Sum float64
        Count int64
    }

Internally these are stored as various "TimeSeries" which is explained below, but basically some list of the basic unit.

When writing "metrics" the StatName is important for the resolution and TTL as well as is UniqueID.

A "total" metric has the form

    TotalMetric{
         Name Statname
         Metric StatMetric
    }


#### When to choose and why

The main writers are

    - cassandra: a binary blob of timeseries points between a time range

        Good for the highest throughput and data effiency for storage

    - cassandra_flat: a row for each time/point

        Good for simplicity, and when you are starting out w/ cassandra to verify things are working as planned

    - whisper: standard graphite format

        Good for local testing or even a local collector for a given node, or if you simply want a better/faster/stronger
        carbon-cache like entity.

    - mysql:

        Good for "slow" stats (not huge throughput or volume as you will kill the DB)


### Writer Schemas

#### MYSQL

Slap stuff in a MySQL DB .. not recommended for huge throughput, but maybe useful for some stuff ..
You should make Schemas like so (`datetime(6)` is microsecond resolution, if you only have second resolution on the
`times` probably best to keep that as "normal" `datetime`).  The TTLs are not relevant here.  The `path_table` is
useful for key space lookups

    CREATE TABLE `{path_table}` (
        `pidth` int NULL,
        `path` varchar(255) NOT NULL DEFAULT '',
        `length` int NOT NULL
        PRIMARY KEY `stat` (`stat`),
         KEY `length` (`length`)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8;

    CREATE TABLE `{table}_{keeperprefix}` (
      `id` int(11) unsigned NOT NULL AUTO_INCREMENT,
      `uid` int NULL,
      `path` varchar(255) NOT NULL DEFAULT '',
      `sum` float NOT NULL,
      `min` float NOT NULL,
      `max` float NOT NULL,
      `first` float NOT NULL,
      `last` float NOT NULL,
      `count` float NOT NULL,
      `resolution` int(11) NOT NULL,
      `time` datetime(6) NOT NULL,
      PRIMARY KEY (`id`),
      KEY `uid` (`uid`),
      KEY `path` (`path`),
      KEY `time` (`time`)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8;

If your for times are `times = ["10s", "1m", "10m"]` you should make 3 tables named

    {tablebase}_10s
    {tablebase}_60s
    {tablebase}_600s

And only ONE path table

    {path_table}

Config Options

    # Mysql
    #  NOTE: this expects {table}_{keepertimesinseconds} tables existing
    #  if timers = ["5s", "10s", "1m"]
    #  tables "{table}_5s", "{table}_10s" and "{table}_60s"
    #  must exist
    [mypregename.accumulator.writer]
    driver = "mysql"
    dsn = "root:password@tcp(localhost:3306)/test"
        [mypregename.accumulator.writer.options]
        table = "metrics"
        path_table = "metrics_path"
        batch_count = 1000  # batch up this amount for inserts (faster then single line by line) (default 1000)
        periodic_flush= "1s" # regardless if batch_count met, always flush things at this interval (default 1s)


#### File

Good for just testing stuff or, well, other random inputs not yet supported
This will dump a TAB delimited file per `times` item of

    statname sum mean min max count resolution nano_timestamp nano_ttl

If your for times are `times = ["10s", "1m", "10m"]` you will get 3 files of the names.

    {filebase}_10s
    {filebase}_60s
    {filebase}_600s


Config Options

    # File
    #  if [keepers] timers = ["5s", "10s", "1m"]
    #  files "{filename}_5s", "{filename}_10s" and "{filename}_60s"
    #  will be created
    #
    # this will also AutoRotate files after a certain size is reached
    [mypregename.accumulator.writer]
    driver = "file"
    dsn = "/path/to/my/filename"
        [mypregename.accumulator.writer.options]
        max_file_size = "104857600"  # max size in bytes of the before rotated (default 100Mb = 104857600)

#### Cassandra

NOTE: there are 2 cassandra "modes" .. Flat and Blob

Flat: store every "time, min, max, sun, count, first, last" in a single row
Blob: store a "chunk" of time (1hour) in a bit packed compressed blob (a "TimeSeries")

Regardless of choice ...

This is probably the best one for massive stat volume. It expects the schema like the MySQL version,
and you should certainly use 2.2 versions of cassandra.  Unlike the others, due to Cassandra's type goodness
there is no need to make "tables per timer".  Expiration of data is up to you to define in your global TTLs for the schemas.
This is modeled after the `Cyanite` (http://cyanite.io/) schema as the rest of the graphite API can probably be
used using the helper tools that ecosystem provides.  (https://github.com/pyr/cyanite/blob/master/doc/schema.cql).
There is one large difference between this and Cyanite, the metrics point contains the "count" which is different
then Cyanite as they group their metrics by "path + resolution + precision", i think this is due to the fact they
dont' assume a consistent hashing frontend (and so multiple servers can insert the same metric for the same time frame
but the one with the "most" counts wins in aggregation) .. but then my Clojure skills = 0.
For consistent hashing of keys, this should not happen.


Please note for now the system assumes there is a `.` naming for metrics names

    my.metric.is.fun


You should wield some Cassandra knowledge to change the on the `metric.metric` table based on your needs
The below causes most compaction activity to occur at 10m (min_threshold * base_time_seconds)
and 2h (`max_sstable_age_days` * `SecondsPerDay`) windows.
If you want to allow 24h windows, simply raise `max_sstable_age_days` to ‘1.0’.

    compaction = {
        'class': 'DateTieredCompactionStrategy',
        'min_threshold': '12',
        'max_threshold': '32',
        'max_sstable_age_days': '1',
        'base_time_seconds': '50'
    }

##### Cassandra Flat Schema

    CREATE KEYSPACE metric WITH replication = {'class': 'SimpleStrategy', 'replication_factor': '3'}  AND durable_writes = true;

    USE metric;

    CREATE TYPE metric_point (
            max double,
            min double,
            sum double,
            first double,
            last double,
            count int
        );


        CREATE TYPE metric_path (
            path text,
            resolution int
        );

        CREATE TABLE metric.metric (
            id varint,
            mpath frozen<metric_path>,
            time bigint,
            point frozen<metric_point>,
            PRIMARY KEY (id, mpath, time)
        ) WITH COMPACT STORAGE
            AND CLUSTERING ORDER BY (mpath ASC, time ASC)
            AND compaction = {
                'class': 'DateTieredCompactionStrategy',
                'min_threshold': '12',
                'max_threshold': '32',
                'max_sstable_age_days': '0.083',
                'base_time_seconds': '50'
            }
            AND compression = {'sstable_compression': 'org.apache.cassandra.io.compress.LZ4Compressor'}
            AND dclocal_read_repair_chance = 0.1
            AND default_time_to_live = 0
            AND gc_grace_seconds = 864000
            AND max_index_interval = 2048
            AND memtable_flush_period_in_ms = 0
            AND min_index_interval = 128
            AND read_repair_chance = 0.0
            AND speculative_retry = '99.0PERCENTILE';

        CREATE TYPE metric.segment_pos (
            pos int,
            segment text
        );


        CREATE TABLE metric.path (
            segment frozen<segment_pos>,
            length int,
            path text,
            id varint,
            has_data boolean,
            PRIMARY KEY (segment, length, path, id)
        ) WITH
             bloom_filter_fp_chance = 0.01
            AND caching = {'keys':'ALL', 'rows_per_partition':'NONE'}
            AND comment = ''
            AND compaction = {'class': 'org.apache.cassandra.db.compaction.SizeTieredCompactionStrategy'}
            AND compression = {'sstable_compression': 'org.apache.cassandra.io.compress.LZ4Compressor'}
            AND dclocal_read_repair_chance = 0.1
            AND default_time_to_live = 0
            AND gc_grace_seconds = 864000
            AND max_index_interval = 2048
            AND memtable_flush_period_in_ms = 0
            AND min_index_interval = 128
            AND read_repair_chance = 0.0
            AND speculative_retry = '99.0PERCENTILE';
        CREATE INDEX ON metric.path (id);

        CREATE TABLE metric.segment (
            pos int,
            segment text,
            PRIMARY KEY (pos, segment)
        ) WITH COMPACT STORAGE
            AND CLUSTERING ORDER BY (segment ASC)
            AND bloom_filter_fp_chance = 0.01
            AND caching = {'keys':'ALL', 'rows_per_partition':'NONE'}
            AND comment = ''
            AND compaction = {'class': 'org.apache.cassandra.db.compaction.SizeTieredCompactionStrategy'}
            AND compression = {'sstable_compression': 'org.apache.cassandra.io.compress.LZ4Compressor'}
            AND dclocal_read_repair_chance = 0.1
            AND default_time_to_live = 0
            AND gc_grace_seconds = 864000
            AND max_index_interval = 2048
            AND memtable_flush_period_in_ms = 0
            AND min_index_interval = 128
            AND read_repair_chance = 0.0
            AND speculative_retry = '99.0PERCENTILE';


### Gotcha's

Some notes from the field::

Write Speed:: Cassandra Protocol v3 (cassandra <= 2.1) is MUCH slower then Protocol v4 (cassandra 2.2 -> 3.X).

Given that we tend to need to write ~100-200 THOUSANDs metric points in our flush window (typically 5s-10s)
if we cannot fully write all the stats in the flush window beteen flush times, the app will have to SKIP a flush write
in order to basically not die a horrible death of RAM consumption and deadlocks.

As a result .. don't use cassandra 2.1, use at least cassandra 2.2

Time Drift :: Golang's concurency and timers are not "realtime" meaning over time (like 30 min) the flush windows will actually
move (i.e. 10s -> 10.00001s -> 10.00003s -> 10.0002s ...) as a result the metrics that get written will start to not be exactly
10s appart, but start to drift from each other.  In Graphite this will cause some "holes" as it expect "exact" 10s offsets, but we need
to interpolate the approximate bin shift windows for graphite to consume.  Golang's `Tickers` do attempt to compensate for drift
but nothing is perfect.  Graphite uses only Seconds for TimeStamps, so all should be pretty well contained in that time bound.

To help with this, the tickers here attempt to try to flush things on proper mod internals
The ticker will try to start on `time % duration == 0` this is not exact, but it usually hits within a few milliseconds of the correct mode.
i.e. a "timer" of "10s" should start on `14579895[0-9]0`

To further make Cassandra data points and timers align, FLUSH times should all be the Smallest timer and timers should be multiples of each other
(this is not a rule, but you really should do it if trying to imitate graphite whisper data), an example config below

    [graphite-cassandra]
    listen_server="graphite-proxy"
    default_backend="graphite-proxy"

    # accumulator and
    [graphite-cassandra.accumulator]
    backend = "graphite-gg-relay"
    input_format = "graphite"
    output_format = "graphite"

    # push out to writer aggregator collector and the backend every 10s
    # this should match the FIRST time in your times below
    accumulate_flush = "10s"

    # aggregate bin counts
    times = ["10s:168h", "1m:720h", "10m:21600h"]

    [graphite-cassandra.accumulator.writer.metrics]
    ...

#### Whisper

Yep, we can even act like good old carbon-cache.py (not exactly, but close).  If you want to write some whisper files
that can be used choose the whisper writer driver.  Unlink the carbon-cache, here only "one set" of aggregate timers
is allowed (just the `times=[...]` field) for all the metrics written (i.e. there is no `storage-schema.cof` built in yet).
Also this module will attempt to "infer" the aggregation method based on the metric name (sum, mean, min, max) rather
then using `storage-aggregation.con` for now.

Aggregation "guesses"  (these aggregation method also apply to the column chosen in the Cassandra/Mysql drivers)

        endsin "mean":              "mean"
        endsin "avg":               "mean"
        endsin "average":           "mean"
        endsin "count":             "sum"
        startswith "stats.count":   "sum"
        endsin "errors":            "sum"
        endsin "error":             "sum"
        endsin "requests":          "sum"
        endsin "max":               "max"
        endsin "min":               "min"
        endsin "upper_\d+":         "max"
        endsin "upper":             "max"
        endsin "lower_\d+":         "min"
        endsin "lower":             "min"
        startswith "stats.gauge":   "last"
        endsin "gauge":             "last"
        endsin "std":               "mean"
        default:                    "mean"


An example config below


    [graphite-whisper]
    listen_server="graphite-proxy"
    default_backend="graphite-proxy"

     # accumulator and
     [graphite-whisper.accumulator]
        backend = "BLACKHOLE"
        input_format = "graphite"
        output_format = "graphite"

     # push out to writer aggregator collector and the backend every 10s
     # this should match the FIRST time in your times below
     accumulate_flush = "10s"

     # aggregate bin counts
     times = ["10s:168h", "1m:720h", "10m:21600h"]

     [graphite-whisper.accumulator.writer.metrics]
        driver="whisper"
     	dsn="/root/metrics/path"
     	xFilesFactor=0.3
     	write_workers=32
     	write_queue_length=102400

     [graphite-whisper.accumulator.writer.indexer]
        driver="whisper"
        dsn="/root/metrics/path"


### KAFKA

I mean why not.  There is no "reader" API available for this mode, as kafka it's not designed to be that.  But you can
shuffle your stats to the kafka bus if needed.  There are 2 message types "index" and "metrics".  They can be
put on the same topic or each in a different one, the choice is yours.  Below is the 2 messages JSON blobs.
You can set `write_index = false` if you want to NOT write the index message (as the metric message has the metric in it
already and consumers can deal with indexing)

        INDEX {
            id: [uint32 MMH3],
    	    type: "index | delete-index",
    	    path: "my.metric.is.good",
    	    segments: ["my", "metric", "is", "good"],
    	    senttime: [int64 unix Nano second time stamp]
    	}

    	METRIC{
    	    type: "metric",
    	    time: [int64 unix Nano second time stamp],
    	    metric: "my.metric.is.good",
    	    id: [uint32 MMH3],
    	    sum: float64,
    	    mean: float64,
    	    min: float64,
    	    max: float64,
    	    first: float64,
    	    last: float64,
    	    count: int64,
    	    resolution: float64,
    	    ttl: int64,
    	    tags: []string //[key1=value1, key2=value2...]
    	}


Here are the configuration options

            [to-kafka.accumulator.writer.metrics]
            driver = "kafka"
            dsn = "pathtokafka:9092,pathtokafka2:9092"
            index_topic = "cadent" # topic for index message (default: cadent)
        	metric_topic = "cadent" # topic for data messages (default: cadent)

        	# some kafka options
        	compress = "snappy|gzip|none" (default: none)
        	max_retry = 10
        	ack_type = "local" # (all = all replicas ack, default "local")
        	flush_time = "1s" # flush produced messages ever tick (default "1s")
        	tags = "server=host1,env=prod" # these are static for whatever process is running this

        	[to-kafka..accumulator.writer.indexer]
            driver = "kafka"
            dsn = "pathtokafka:9092,pathtokafka2:9092"

        	    [to-kafka..accumulator.writer.indexer.options]
                write_index = false|true

If you want to bypass the entire "graphite" thing and go straight to a kafka dump, look to
`configs/statsd-kafka-config.toml` and `configs/statsd-kafka-pregre.toml` pair .. this is probably the most typical use
of a kafka writer backend.  One can easily do the same with straight graphite data (i.e. from `diamond` or similar).


Since ordering and time lag and all sorts of other things can mess w/ the works for things, it's still best to fire stats to
a consistent hash location, that properly routes and aggregates keys to times such that stats are emitted "once" at a given time
window.  In that way, ordering/time differences are avoided.  Basically  `statsd -> flush to consthasher -> route -> flush to kafka`
