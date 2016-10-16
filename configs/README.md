

Example Cadent Configs
======================

This simply contains a bunch of common example configs for certain backends and inputs

## config

The basic mode for cadent is a consistent hashing relay which.

    cadent --config=configs/graphite/graphite-relay.toml


## prereg

`prereg` is the accumulators/writers/api section of cadent.

nameing is as follows .. each "input" type is put into a folder (statsd, carbon2, graphite)

{input}-{writer}-{type}.toml

where

{input} is "graphite", "statsd", "carbon2"

{writer} is "mysql", "cassandra", "elasticsearch", "kafka", "whisper", "relay"

    the "relay" is simply using cadent as a consistent hash ring fowarder.

{type} is "series" or "flat" (or in the case of pure relay, the output format "graphite", etc)


To run

    cadent --config={input}-config.toml --prereg={input}-{writer}-{type}.toml


## API Only mode (alpha)

The "apionly" directory has a config for just operating the API w/o all the writer stuff

Please note this API Only mode is in progress, It will easily read from the backend DBs, but if using
the Series type for your metrics on different writers, the mechanism to pull from the relevent write-back caches
is not yet complete.


    cadent --api=configs/apionly/api.toml

