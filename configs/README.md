

Example Cadent Configs
======================


This simply contains a bunch of common example configs for certain backends and inputs

nameing is as follows .. each "input" type is put into a folder (statsd, carbon2, graphite)

{input}-{writer}-{type}.toml

where

{input} is "graphite", "statsd", "carbon2"

{writer} is "mysql", "cassandra", "elasticsearch", "kafka", "whisper", "relay"

    the "relay" is simply using cadent as a consistent hash ring fowarder.

{type} is "series" or "flat" (or in the case of pure relay, the output format "graphite", etc)


To run

cadent --config={input}-config.toml --prereg={input}-{writer}-{type}.toml


