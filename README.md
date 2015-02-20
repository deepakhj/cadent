

conshashsrv
===========

Generic Consistent hash Server in Go

Installation
------------

    Well, first you need to install go .. https://golang.org

    git clone git@scm-main-01.dc.myfitnesspal.com:infra/consthashsrv.git
    cd consthashsrv/cmd/consthash
    
    // because things are "private" we need to "hack around" go's usual pulling/get mechanism
    //pull in more git repos you'll need
    git clone git@scm-main-01.dc.myfitnesspal.com:goutil/statsd.git
    git clone git@scm-main-01.dc.myfitnesspal.com:goutil/consistent.git
     
    cd ../../
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
We can make a fancy status page when nesesary

    localhost:6061/stats

It will also emit it's owns stats to statsd as well

Author
------

boblanton@myfitnesspal.com 2015 MyFitnesspal