

conshashsrv
===========

Generic Consistent hash Server in Go

Installation
------------

    Well, first you need to install go .. https://golang.org  1.5.1

    git clone git@scm-main-01.dc.myfitnesspal.com:infra/consthashsrv.git
    cd consthashsrv/server
    
    // because things are "private" we need to "hack around" go's usual pulling/get mechanism
    //pull in more git repos you'll need
    git clone git@scm-main-01.dc.myfitnesspal.com:goutil/statsd.git
    git clone git@scm-main-01.dc.myfitnesspal.com:goutil/consistent.git
     
    cd ../../
    make
   

Examples
--------

Look to the "example-config.toml" for all the options you can set and their meaning

to start

    consthash --config=example-config.toml
    
There is also the "PreReg" options files as well, this lets one pre-route lines to various backends, that can then 
be consitently hashed, or "rejected" if so desired

    consthash --config=example-config.toml --prereg=example-prereg.toml


What I do
---------

This was designed to be an arbitrary proxy for any "line" that has a representative Key that needs to be forwarded to
another server consistently (think carbon-relay.py in graphite and proxy.js in statsd) but it's able to handle
any other "line" that can be regex out a Key.  

It Supports health checking to remove nodes from the hash ring should the go out, and uses a pooling for
outgoing connections.  It also supports various hashing algorithms to attempt to imitate other
proxies to be transparent to them.

Replication is supported as well to other places

The Flow of a given line looks like so

    InLine -> Listener -> PreReg -> Backend -> Hasher -> outLine
                                      |
                                      -> Replicator -> Hasher -> outLine

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

things come with an "echo server" which is simply what it is .. just echos what it gets to stdout

the make will make that as well, to run and listen on 3 UDP ports

    echoserver --servers=udp://127.0.0.1:6002,udp://127.0.0.1:6003,udp://127.0.0.1:6004
    
    # 3 tcp servers
    echoserver --servers=tcp://127.0.0.1:6002,tcp://127.0.0.1:6003,tcp://127.0.0.1:6004
    

You may want to get a statsd deamon (in golang as well) .. (github.com/bitly/statsdaemon)

    statsdaemon -debug -percent-threshold=90 -percent-threshold=99  -persist-count-keys=0 -address=":8136" -admin-address=":8137"  -receive-counter="go-statsd.counts" -graphite="127.0.0.1:2003"

There is also a "line msg" generator "statblast"

   
    Usage of statblast:
      -buffer int
            send buffer (default 512)
      -forks int
            number of concurrent senders (default 2)
      -rate string
            fire rate for stat lines (default "0.1s")
      -servers string
            list of servers to open (tcp://127.0.0.1:6002,tcp://127.0.0.1:6003) (default "tcp://127.0.0.1:8125")
      -type string
            statsd or graphite (default "statsd")




Author
------

boblanton@myfitnesspal.com 2015 MyFitnesspal