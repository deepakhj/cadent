

conshashsrv
===========

Generic Consistent hash Server in Go

Installation
------------

    Well first you need to install go .. https://golang.org

    git clone git@scm-main-01.dc.myfitnesspal.com:infra/consthashsrv.git
    cd consthashsrv
    make
   

Examples
--------

Look to the "config.toml" for all the options you can set and their meaning

to start

    consthash --config=config.toml

What I do
---------

This was designed to be an arbitrary proxy for any "line" that has a representative Key that needs to be forwarded to
another server consistently (think carbon-relay.py in graphite and proxy.js in statsd) but it's able to handle
any other "line" that can be regex out a Key.  

It Supports health checking to remove nodes from the hash ring should the go out, and uses a pooling for
outgoing connections.  It also supports various hashing algorithms to attempt to imitate other
proxies to be transparent to them.

It runs it's own micro stats server so you can ping it for it's internal stats (very lightweight at the moment)

    localhost:6061/stats

It will also emit it's owns stats to statsd as well

Author
------

boblanton@myfitnesspal.com 2015 MyFitnesspal