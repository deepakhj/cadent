

Cadent
======

Manipulate your metrics

So in terms of picking a bad name ... marketers .. is "k-dent" as in "k as in cake" \ˈkā-dənt\ .. 
call it "ca-dent" (ca as in cat) if you really want to .. but that holds no real meaning (just look it up ... expand your vocab)

Basically this acts like 4-6 existing projects out in the wilderness statsd, statsd-proxy, carbon-relay, carbon-aggegator, carbon-cache, cyanite

It "sorta" behaves like carbon-cache, (as it can read/write to whisper files but does not have the "entire functionality set" of carbon-cache)
Cassandra, and even a SQL DB if you really think that's a good idea, watch as you kill your RDBS trying to insert 100k items in 1 second)

But dramatically retooled and optimized to handle 100's of thousands of metrics a second.

Has a Hekad feel (same sort of data pipeline like) but optimized for what we really need

 - consistent hashing relay
 - metric filter and router
 - relay replication
 - accumulation and aggregation
 - time series DB writing
 - graphite-api endpoints
 
Yes "each" of the above can be handled by a standalone app (and they do exist in the echo system) .. however at the volume
we want to address .. there's nothing like raw RAM/CPU power on a local node (the internet, slow, ram, fast).

Note:: configuration for complex scenarios of loop backs, replicas moving, multiple backends, accumulators, and aggregators 
can get confusing .. and you can find yourself hitting yourself over the head alot.  You's say "why not keep it simple"
If metrics collection and manipulating was simple, I would not have to write this.


Installation
------------

    Well, first you need to install go .. https://golang.org  >= 1.5.1
    And a kernel that supports SO_REUSEPORT (linux 3.9 or higher and bsd like kernels)
    And make sure to use VENDOR support (in golang 1.6, this is the default)
    

    git clone git@scm-main-01.dc.myfitnesspal.com:Metrics/cadent.git
    export GOPATH=$(pwd)

    make
   

Examples
--------

Look to the "example-config.toml" for all the options you can set and their meaning
and config directory for more examples.

to start

    cadent --config=example-config.toml
    
There is also the "PreReg" options files as well, this lets one pre-route lines to various backends, that can then 
be consitently hashed, or "rejected" if so desired, as well as "accumulated" (ala statsd or carbon-aggrigate)

    cadent --config=example-config.toml --prereg=example-prereg.toml


What I do
---------

Once upon a time, our statsd servers had __waaaayyyy__ to many UDP errors (a common problem i've been told). 
Like many projects, necessity, breeds, well, more necessity, but that's another conversation.
So a Statsd Consistent Hashing Proxy in Golang was written to be alot faster and not drop any packets.
 
And then ...

This was designed to be an arbitrary proxy for any "line" that has a representative Key that needs to be forwarded to
another server consistently (think carbon-relay.py in graphite and proxy.js in statsd) but it's able to handle
any other "line" that can be regex out a Key.  

And then ...

It Supports health checking to remove nodes from the hash ring should the go out, and uses a pooling for
outgoing connections.  It also supports various hashing algorithms to attempt to imitate other
proxies to be transparent to them.

And then ...

Replication is supported as well to other places, there is "alot" of polling and line buffereing so we don't 
waste sockets and time sending one line at a time for things that can support multiline inputs (i.e. statsd/graphite)

And then ...

A "PreRegex" filter on all incoming lines to either shift them to other backends (defined in the config) or
simply reject the incoming line

And then ...

Finally an Accumulator, which initially "accumulates" lines that can be (thing statsd or carbon-aggrigate) then 
emits them to a designated backend, which then can be "PreRegex" to other backends if nessesary
Currently only "graphite" and "statsd" accumulators are available such that one can do statsd -> graphite or even 
graphite -> graphite or graphite -> statsd (weird) or statsd -> statsd.  The {same}->{same} are more geared
towards pre-buffering very "loud" inputs so as no to overwhelm the various backends.

The Flow of a given line looks like so

    InLine(s) -> Listener -> Splitter -> [Accumulator] -> [PreReg/Filter] -> Backend -> Hasher -> OutPool -> Buffer -> outLine(s)
                                                                |
                                                                [-> Replicator -> Hasher -> OutPool -> Buffer -> outLine(s)]
Things in `[]` are optional

NOTE :: if in a cluster of hashers and accumulators .. NTP is your friend .. make sure your clocks are in-sync

What Needs Work
---------------

Of course there are always things to work on here to improve the world below is a list that over time i think should be done

1. Bytes vs Strings: strings are "readonly" in goland and we do alot of string manipulation that should be memory improved by using []bytes instead.
Currently much of the system uses strings as it's basis for things, which was a good choice intially (as all the stats/graphite
lines are strings in the incoming) but at large loads this can lead to RAM and GC issues.

2. OverFlow handling: there are ALOT of buffers and internal caching and compression of timeseries going on to help w/
metric floods, RAM pressure, writing issues and so on.  There is no real magic bullet i've found yet to be able to handle huge loads w/o dropping
some points in the series at somepoint in the chain.  Things like backpressure are hard to impliment as the senders need to
also understand backpressure (which UDP cannot) and various stats sending clients do not either (as it requires them to
also have some interal buffering mechanisms when issues occur).

Personally, the best mechanism may be to imitate a cassandra/kafka append only rotating log on the file system, which then
the writers/indexers simply consume internally to do the writes. Much similare to how Hekad's ElasticSearch writer behaves
but, even w/ this mechanism, if the writers cannot keep up eventually there will be death. Internally the "write-cache" is this
mecahnism of a sort, in RAM, but will simply drop overflowing points.

3. Metric Indexing: Graphites "format" was made for file glob patterns, which is good if everything is on a file system
this is not the case for other data stores.  And intertroducing a "tag/multi-dim" structure on top of this just makes
indexing that much harder of a problem to be able to "find" metrics you are looking for (obviously if you know the exact
name for things, this is easy, ans some TSDBs do indeed force this issue), but we are spoiled by the graphite's finder abilities.

4. API Reads: Internally there are 3 different "reads" for every single "read" request.
The internal "write cache", an LRU "read cache" and the "data store" itself.  There are fragments of the time series at any given
time in each of these 3 places.  So a given read request (especially for "recent-ish" data) will hit all 3.  Merging them can
be a bit of an expensive operation.  Ideally most reads will eventually end up just hitting the read-cache (which is a
 smart cache that will get points auto-added as the come in if the metric has been requested before even before writing).

## Accumulators 

Accumulators almost always need to have the same "key" incoming.  Since you don't want the same stat key accumulated
in different places, which would lead to bad sums, means, etc.  Thus to use accumulators effectively in a multi server
endpoint scenario, you'd want to consistently hash from the incoming line stream to ANOTHER set of listeners that are the 
backend accumulators (in the same fashion that Statsd Proxy -> Statsd and Carbon Relay -> Carbon Aggregator).  


It's easy to do this in one "instance" of this item where one creates a "loop back" to itself, but on a different
listener port.

     InLine(port 8125) -> Splitter -> [PreReg/Filter] -> Backend -> Hasher -> OutPool (port 8126)
        
        --> InLine(port 8126) -> Splitter -> [Accumulator] -> [PreReg/Filter] -> Backend -> OutPool (port Other)

This way any farm of hashing servers will properly send the same stat to the same place for proper accumulation.

### Time 

A finicky thing.  So it's best to have NTP running on all the nodes.  

I don't claim nanosecond proper timescales yet this requires much more syncro of clocks between hasher nodes then
this currently has, but milliseconds should work so long as NTP is doing it's job.

For protocals that don't support a "time" in their line protocal (statsd), time is "NOW" whenever time is needed.
(i.e. a statsd incoming to graphite outgoing). 

For those that do (graphite), time will be propogated from whatever the incoming time is.  Since things are "binned"
by some flush/aggregation time, any incomming will be rounded to fit the nearest flush time and aggregated in that
bin.  The Binning is effectively the nearest `time % resolution` (https://golang.org/pkg/time/#Time.Truncate)

For Regex lines, you can specify the `Timestamp` key for the regex (if available) same as the `Key`. i.e
 
    `^(<\d+>)?(?P<Timestamp>[A-Z][a-z]+\s+\d+\s\d+:\d+:\d+) (?P<Key>\S+) (?P<Logger>\S+):(.*)`

To use it properly you will need to specify a `timeLayout` in the regex options, of the golang variety
(see: https://golang.org/src/time/format.go .. note that the timeLayout should be a string example like shown
"Mon Jan 02 15:04:05 2006")

Time really only matters for sending things to writer backends or another service that uses that time.

*Please note::* certain backends (i.e. cassandra, mysql, etc) will "squish" old data points if times are sent in 
funny order.  Meaning if you send data with times 1pm, 1pm, 1pm in one "flush period" then all the 3 of those "1pm times"
will get aggregated and inserted in the DB.  However, if one tries to "re-add" the 1pm data sometime in the future 
this will clobber the old data point with the new data.  Meaning that we don't aggregate the old value w/ a new one
as this particular process is _very_ expensive (select if exists, merge, update) vs just upsert.  When attempting
to write 200k/points sec, every milliseconds counts.  If this is something that's a requirement for you, please let
me know and i can try to make it a writer option, but not as the default.  

Unlike the generic `graphite` data format, which can have different time retentions and bin sizes for different metrics
I have taken the approach that all metrics will have the same bin size(s).  Meaning that all metrics will get 
binned into `times` buckets that are the same (you can have as many as you wish) and to keep the math simple and fast
the timer buckets should be multiples of each other, for instance.

    times = ["5s", "1m", "10m"] 
    
OR w/ TTLs (note must be in "hours/min/seconds" not days/months)

    times = ["5s:168h", "1m:720h", "10m:17520h"] 
    
This also means the "writers" below will need to follow suit with their data stores and TTLs,  Things like MySQL and files
have no inherent TTLs so the TTLs are not relevant and are ignored, Cassandra, on the other hand, can have these TTLs per item. Also
TTLs are outed as "time from now when i should be removed", not a flat number.  

THe "base" accumulator item will constantly Flush stats based on the first time given (above every `5s`). It is then Aggregators
caches to maintain the other bins (from the base flush time) and flush to writers at the appropriate times. 

_MULTIPLE TIMES ONLY MATTER IF THERE ARE WRITERS._

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

The StatName has a "UniqueID" which is basically a FNV-64a hash of the following

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


### TimeSeries

The core of things for writers (not really used at all in the simply Constist Hashing or Statsd modes).

There are a number of ways for store things.  Some are very good a RAM compression and others are good for ease of use
compatibility, and other internal uses as explained below.

Some definitions:

    - DeltaOfDeltas:

        Since 99.9% of the time "Time" is moving foward there is not need to store the full "int64" each time
        So we store a
        "start" time (T0),
        the next value is then T1 - T0 =  DeltaT1,
        the next value is then T2 - T1 = DeltaT2

        To get a time a the "Nth" point we simply need to

        TN = T0 + Sum(DeltaI, I -> {1, N})

    - VarBit:

        A "Variable Bit encoding" which will store a value in the smallest acctual size it can be stored int

        If the type is an "int64" but the value is only 0 or 1, just store one byte (plus some bits to say what it was)
        if the value is 1000, then only store 2 bytes,
        etc.

        If the type is a float64, but the value is just an int of some kind it will use the int encodings above

    - "Smart"Encoding:

        You'll notice that we store 7 64 bit numbers in the StatMetric.  Sometimes (alot of times) we only have
        a Count==1 in the above which means that all the floats64 are the same (or they better be).  If that's the
        case, we only store one float value (the Sum) (and, depending on the format below, a bit that tells us this fact)

    - TimeResolution:

        Some series (gob, protobuf, gorilla) can use the "int32" for seconds (lowres) of time and
        not int64 (highres) for nanosecond resolution.
        Lowres is the "default" behavior as most of the time our flush times are in the SECONDS not less


#### GOB + DeltaOfDeltas + VarBit + SmartEncoding + TimeResolution

The gob format which is a GoLang Specific encoding https://golang.org/pkg/encoding/gob/ that uses the VarBit encoding internally.

This method is pretty efficent.  But it is golang specific so it's not too portable.  But may be good for other uses
(like quick file appenders or something in the future).

#### ProtoBuf + VarBit + SmartEncoding + TimeResolution

Standard protobuf encoder github.com/golang/protobuf/proto using the https://github.com/gogo/protobuf generator

Since protobuf is pretty universal at this point (lots of lanuages can read it and use it) it's pretty portable
It's also a bit more efficent in space as the GOB form, due to the nicer encoding methods provided by gogo/protobuf

Also this is NOT time order sensitive, it simply stores each StatMetric "as it is" and it's simple an array
of these internally, so it's good for doing slice operations (i.e. walking time caches and SORTING by times)

This uses "gogofaster" for the generator of proto schemas

    go get github.com/gogo/protobuf/protoc-gen-gogofaster
    protoc --gogofaster_out=. *.proto


#### Json

The most portable format, but also the biggest (as we a storing strings in reality).  I'd only use this if you
need extream portability across things, as it's not really space efficent at all.

#### Gorilla + DeltaOfDeltas + VarBit + SmartEncoding + TimeResolution

The most extream compression available.  As well as doing the same goodies mentioned, the core squeezer is the
Float64 compression it uses.  Read up on it here http://www.vldb.org/pvldb/vol8/p1816-teller.pdf.

This does NOT even remotely support out-of-time-order inputs.  The encoding interal to Cadent is a modified version
that allows for multi float encodings, Nano-second encodings and the "smart" encoding as well.

This is by far the best format to store things if your pieces can support it (both in ram and longterm), but due to
the forced timeordering, lack  of sorting.  It does not play well w/ many internal things.

The compression is also highly variable depending on incoming values, so it can be hard to "know" what storage or ram
constraints will be needed a-priori (unless you know the domain of your metrics well).


#### Repr

This is the "native" internal format for a metric cadent.  It's NOT very space conciderate (as it's vewry similare
to the json format, but w/ more stuff).  Basically don't use it for storage.


#### ZipGob  + DeltaOfDeltas + VarBit + SmartEncoding + TimeResolution

Instead of a flat []byte buffer for gob encoding use the FLATE buffer (otherwise exactly the same as Gob).  While it
does some some space, due to the internals of the golang Flate, there is alot of GC churn and evils associated
with this one if used for a large number of series.  So this may be a good "final persist state" but it should get converted
out of this format for use elsewhere.


### Writer Schemas

#### MYSQL

Slap stuff in a MySQL DB .. not recommended for huge throughput, but maybe useful for some stuff ..
You should make Schemas like so (`datetime(6)` is microsecond resolution, if you only have second resolution on the 
`times` probably best to keep that as "normal" `datetime`).  The TTLs are not relevant here.  The `path_table` is 
useful for key space lookups
    
    CREATE TABLE `{path_table}` (
        `path` varchar(255) NOT NULL DEFAULT '',
        `length` int NOT NULL
        PRIMARY KEY `stat` (`stat`),
         KEY `length` (`length`)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8;

    CREATE TABLE `{table}_{keeperprefix}` (
      `id` int(11) unsigned NOT NULL AUTO_INCREMENT,
      `stat` varchar(255) NOT NULL DEFAULT '',
      `sum` float NOT NULL,
      `min` float NOT NULL,
      `max` float NOT NULL,
      `first` float NOT NULL,
      `last` float NOT NULL,
      `count` float NOT NULL,
      `resolution` int(11) NOT NULL,
      `time` datetime(6) NOT NULL,
      PRIMARY KEY (`id`),
      KEY `stat` (`stat`),
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


    CREATE TYPE metric_id (
        path text,
        resolution int
    );

    CREATE TABLE metric.metric (
        id frozen<metric_id>,
        time bigint,
        point frozen<metric_point>,
        PRIMARY KEY (id, time)
    ) WITH COMPACT STORAGE
        AND CLUSTERING ORDER BY (time ASC)
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
        path text,
        length int,
        has_data boolean,
        PRIMARY KEY ((segment, length), path)
    ) WITH CLUSTERING ORDER BY (path ASC)
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
            id: [int64 FNV64a],
    	    type: "index | delete-index",
    	    path: "my.metric.is.good",
    	    segments: ["my", "metric", "is", "good"],
    	    senttime: [int64 unix Nano second time stamp]
    	}
    	
    	METRIC{
    	    type: "metric",
    	    time: [int64 unix Nano second time stamp],
    	    metric: "my.metric.is.good",
    	    id: [int64 FNV64a],
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
            

### API/Readers

Readers are an attempt to imitate the Graphite API bits and include 3 main endpoints

    /{root}/find  -- find paths ( ?query=path.*.to.my.*.metric )
    /{root}/expand -- expand a tree ( ?query=path.*.to.my.*.metric )
    /{root}/metrics -- get the actuall metrics ( ?target=path.*.to.my.*.metric&from=time&to=time )

Unlike the Whisper file format which keeps "nils" for no data (i.e. a round robin DB with a fixed step size and known counts),
the mature of the metrics in our various backends write points at what ever the flush time is, and if there is nothing to write
does not write "nils" so the `/metrics` endpoint has to return an interpolated set of data to attempt to match what graphite expects
(this is more a warning for those that may notice some time shifting in some data and "data" holes)

This may mean that you will see some random interspersed `nils` in the data on small time ranges.  There are a variety of reasons for this
1) flush times are not "exact" go's in the concurency world, not everything is run exactly when we want it do so over time, "drift" will 
2) Since we are both "flushing to disk" and "flushing from buffers" from many buffers at different times, sometimes they just don't line up

*NOTE*  Currently only Cassandra and Whisper "readers" are valid (MySQL can be, needs the code to be written). File and Kafka writers can have no reader apis.

#### Cassandra

Whisper and Cassandra are currently the only "readers" available, configured in the PreReg `Accumulator` section as follows

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



### Listen Server Types

All inputs and out puts can be tcp, udp, unix socket, or http

    tcp -> tcp://127.0.0.1:2003
    udp -> udp://127.0.0.1:8125
    unix -> unix:///my/socket.sock
    http -> http://moo.org/stats
    

http expects the BODY of the request to basically be the lines

    GET /stats HTTP/1.1
    Host: moo.org
    Accept: */*
    
    key value thing
    key value thing2

There is also a special `listen` called `backend_only` which is simply a place where things can routed to internally
(from say a `PreReg` filter or `Accumulator`) that then out puts to it's consthash server outgoing list
    
### Input line types for no accumulator
 
    Graphite: <key> <value> <time>
    Statsd: <key>:<value>|<type>|<samplerate>
    Regex: (<\d+>)?(?P<Timestamp>[A-Z][a-z]+\s+\d+\s\d+:\d+:\d+) (?P<Key>\S+) (?P<Logger>\S+):(.*)

 - note: regex need a `(?P<Key>...)` group to function as that will be the hashing key, other fields are ignored

### Input line types for using accumulator

    Graphite: <key> <value> <time>
    Statsd: <key>:<value>|<type>|<samplerate>


### Internal Stats

It runs it's own micro stats server so you can ping it for it's internal stats (very lightweight at the moment)
We can make a fancy status page when necessary

    # for a little interface to the stats
    
    localhost:6061/

    # the json blob stats for the above html file
    
    localhost:6061/stats
    
    # if you want just a specific server set
    
    localhost:6061/echo-example/stats

It will also emit it's owns stats to statsd as well using a buffered internal statsd client (https://gitlab.mfpaws.com/goutil/statsd).


#### Handy "what are the URL" reference (because i always forget myself)

    http://localhost:6060/servers


### Status

If you have some checker (say nagios) and you want to get the helath status of the server itself

    localhost:6061/ops/status
    
IF you want a specific exerver set

    localhost:6061/echo-example/ops/status


### Check Keys/Server pairs

You can "check" what server a given "key/line" will go to as well using a little json GET url

    localhost:6061/hashcheck?key=XXXXXX
    
This will dump out the server this key will go to and the actuall "hash" value for it for all various running hash servers


### Add/Remove hosts dynamically

You can also Add and remove servers from the hashpool via a POST/PUT

    curl localhost:6061/echo-example/addserver --data "server=tcp://127.0.0.1:6004"
    
     PARAMS:
     
     - server: url : url of the host to add (udp/tcp)
     - check_server: url: url of the server to "check" for the above server that it is alive (defaults to `server`)
     - hashkey: string: the "string" that is the KEY for the hash algo (see `hashkeys` in the TOML config)
     - replica: int: the index of the replica (if any) for a server definition (defaults to 0)
    
    curl localhost:6061/echo-example/purgeserver --data "server=tcp://127.0.0.1:6004"
    
    PARAMS:
    
    - server: url : url of the host to add (udp/tcp)
    - replica: int: the index of the replica (if any) for a server definition (defaults to 0)
            
   
here `echo-example` is the name of the toml server entry 


    
Testing and Dev
---------------

Some quick refs for performance and other "leaky/ram" usages for tuning your brains
(we've the profiler stuff hooked)

    
    go tool pprof  --inuse_space --nodefraction=0.0001 --web  http://localhost:6060/debug/pprof/heap
    go tool pprof  --inuse_objects --nodefraction=0.0001 --web  http://localhost:6060/debug/pprof/heap
    go tool pprof  --alloc_space --nodefraction=0.0001 --web  http://localhost:6060/debug/pprof/heap
    go tool pprof  --alloc_objects --nodefraction=0.0001 --web  http://localhost:6060/debug/pprof/heap
    go tool pprof --nodefraction=0.0001 --web  http://localhost:6060/debug/pprof/profile
    
    
    # just how many mutex locks are stuck?
    curl http://127.0.0.1:6061/debug/pprof/goroutine?debug=2
    

things come with an "echo server" which is simply what it is .. just echos what it gets to stdout

the make will make that as well, to run and listen on 3 UDP ports

    echoserver --servers=udp://127.0.0.1:6002,udp://127.0.0.1:6003,udp://127.0.0.1:6004
    
    # 3 tcp servers
    echoserver --servers=tcp://127.0.0.1:6002,tcp://127.0.0.1:6003,tcp://127.0.0.1:6004
    

There is also a "line msg" generator "statblast." It will make a bunch of random stats based on the `-words` given

   
    Usage of statblast:
      -buffer int
        	send buffer (default 512)
      -forks int
        	number of concurrent senders (default 2)
      -rate string
        	fire rate for stat lines (default "0.1s")
      -servers string
        	list of servers to open (tcp://127.0.0.1:6002,tcp://127.0.0.1:6003), you can choose tcp://, udp://, http://, unix:// (default "tcp://127.0.0.1:8125")
      -type string
        	statsd or graphite (default "statsd")
      -words string
        	compose the stat keys from these words (default "test,house,here,there,badline,cow,now")
      -words_per_stat int
        	make stat keys this long (moo.goo.loo) (default 3)


## Performance

After many days a'tweaking and finding the proper ratios for things here are some tips.  But by all means please tweak
No system is the same, and you will run into context locking and general kernel things at crazy volumes.  

For UDP and TCP reading connections, SO_REUSEPORT is in use, which means we bind multi listeners to the same
address/port .. which means we leave lots to the kernel to handle multiplexing (it being C and low level goodness, 
guess what, performs mucho better).  The `workers` below are the number of bound entities to the socket. 
`out_workers` are the number of dispatch works to do writes to output buffers.


### For "reading" UDP incoming lines

    num_procs = N             # number of cores to use
    workers = N*4             # processes to consume the lines .. this you can tweak depending on your system
    out_workers = N * 8       # dispatchers to deal with output .. this you can tweak depending on your system
    
### For "reading" TCP incoming lines

    num_procs = N             # number of cores to use
    workers = N*4             # processes to consume the lines .. this you can tweak depending on your system
    out_workers = N * 8       # dispatchers to deal with output .. this you can tweak depending on your system


### For "reading" HTTP incoming lines

TBD

### For "reading" Socket incoming lines

TBD


### Whisper Writing

99% of the performance issue w/ Wisper files are the Disks.  Since we typically here have large space requirements
(in the TB range) and we are in the Cloud (AWS for us).  We need to use EBS drives which are really slow compared
to any SSDs in the mix.  So you MUST limit the `writes_per_second` allowed or you'll end up in IOWait death.  For a
1 Tb (3000 iops) generic EBS (gp2) drive empirically we find that we get ~1500 batch point writes per/second max 
(that's using all the IOPs available, which if you think of each "write" as needing to read first then write that 
makes some sense).  So we set the `writes_per_second=1200` to allow for readers to actually function a bit.


### Writers Cache/Ram

This one is a bit tricky to figure out exactly, and it highly dependent on the metric volume and shortest "tick" interval.
The cache ram needed depends on # metrics/keys and the write capacity.  The cache holds a `map[key][]points`.  Once
the writer determines which metric to flush to disk/DB we reclaim the RAM.

Just some empirical numbers to gauge things, but the metric you should "watch" about the ram consumed by the cache are 
found in `stats.gauges.{statsd_prefix}.{hostname}.cacher.{bytes|metrics|points}`.

    Specs: 
        Instance: c4.xlarge
        EBS drive: 1TB (3000 IOPS)
        Flush Window: 10s
        Keys Incoming: 140,000
        Writes/s: 1200(set) ~1000 (actual)
        CacheRam consumed: ~300-600MB
        # Points in Cache: ~1.3 Million

The process that runs the graphite writer then consumes ~1.2GB of ram in total.  Assuming the key space does not
increase (by "alot") the above is pretty much a steady state.


Author
------

![A Flow of examples](configs/example-flow.png)

boblanton@myfitnesspal.com 2015-2016 MyFitnesspal