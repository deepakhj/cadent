

# LineFormats

Currently allowed incoming line formats, all you need to specify is

    `msg_type="<format>"`

## Graphite

`msg_format="graphite"`

    <key> <value> <time> name=value name=value ...

## Statsd

`msg_format="statsd"`

    <key>:<value>|<type>|<samplerate>|#name:value,name:value

## Carbon2

`msg_format="carbon2"`


All of the below are accetible

    name=val name=val name=val  <meta_tag> <value> <time>
    name=value.name=value  <meta_tag> <value> <time>
    name=value,name=value  <meta_tag> <value> <time>
    name_is_value.name_is_value  <meta_tag> <value> <time>


    - has TWO SPACES between the <tag> and <meta_tag> section

    - Don't put `.|,|=|\s` in the value of a tag (or any other punctuation really, if you need punctuation
      in a tag, you're doing something wrong.

    - <meta_tags> are NOT concidered to signify a different metric

    - the "<key>" we unique idenfity a metric is a SORTED by name string of the form

        `nameA_is_value.nameB_is_value.nameC_is_value`

    - there are some "fixed" "intrinsict" tags defined for the metrics2.0 layout, these tags
      are used to determin the <key> and unique ID of a given Stat


        "host"
        "http_method"
        "http_code"
        "device"
        "unit"
        "what"
        "type"
        "result"
        "stat"
        "bin_max"
        "direction"
        "mtype"
        "unit"
        "file"
        "line"
        "env"
        "dc"
        "zone"


    - all other tags are concidered "metatags" and not included in the Unique identifier of a given metric
    - there are 2 required "tags" for Metrics2.0 to be "proper" `mtype` and `unit`

        mytype = rate|count|counter|gauge|timestamp
        unit = (k|M|T...)B|jiff|Hz|...

    - the `stat` tag is used for aggregation choosing, if none is present `mean` is the default agg method

        stat = max|min|std|mean|sum|upper_\d+|lower_\d+|min_\d+|max_\d+|count_\d+|median|median_\d+


## Regex

`msg_format="regex"`


basically whatever you need, but must include a "<Key>" field

    (<\d+>)?(?P<Timestamp>[A-Z][a-z]+\s+\d+\s\d+:\d+:\d+) (?P<Key>\S+) (?P<Logger>\S+):(.*)

 - note: regex need a `(?P<Key>...)` group to function as that will be the hashing key, other fields are ignored.

 The regex field type is ONLY used for any consitent hashing on the incoming line.  "Writers" have nothing to do
 (yet) w/ this style.


## OpenTSDB


`msg_format="opentsdb"`


Their line format is the exact same as graphite's execpt for 4 characters

    put <key> <value> <time> <tag> ...

So all we really do here is "remove" the `put ` and foward it to a graphite line item.  But due to that, it needs
to be known that it is a opentsdb line.


## Json

`msg_format="json"`

Again much like the OpenTSDB json format,  however, if you are sending this via a "line" protocal, one had best be sure
the json blob is ON ONE LINE.

As a result there is a http handler for these json formats

    [json-http]
    listen="http://0.0.0.0:2004/moo"
    msg_type="json"

This will have a route `http://host:2004/moo/json` which you can post JSON of the form

* NOTE THE EXTRA `/json` on the URL *

Single Metric

    {
        metric: xxx,
        value: 123,
        timestamp: 123123,
        tags: {
            moo: goo,
            foo: bar
        }
    }


Or many metrics

    [
       {
            metric: xxx,
            value: 123,
            timestamp: 123123,
            tags: {
                moo: goo,
                foo: bar
            }
        },
        ...
    ]

If you use the tcp line item Or the raw http input route (in the above case `http://host:2004/moo/`) you best "squeeze"
that json to one line, and ths one does not accept "lists"


    [json-tcp]
    listen="tcp://0.0.0.0:2004"
    msg_type="json"


Only the single line is allowed for TCP based inputs or the "raw" http://host:port/


    {"metric":"xxxx","value":123,"timestamp":14123123,"tags":{"moo":"goo", "foo":"bar"}}


You can also use the UDP accepter, but given the length/size of the metrics in this format, it's not recommended.
