package prereg

/*

Parse a Preliminary Regex (or static) Key to backend mapping

Kinda like the carbon-relay where you could do regex redirects on key values, but farms it
internally to the main server bits that then does ConstHashing

This allows keys comming in to be assigned to backend for the const hasher
so that you can say have something that does

NOTE: the the regex will be over the ENTIRE line for a regex based one
for statsd and graphite the key field for each

regex: <key> # aka everything
graphite: <key> <value> <time> <thigns>
statsd: <key>:<data>

graphite for diamond based stats from all database that spans 3 machines (and hashed between them)
graphite for diamond based stats for all app servers that spans 3 machines (and hashed between them)

loglines that look like java ones to be hashed to 5 nodes
loglines that look like rails ones to be hashed to 5 nodes
etc


Should be of the form

the backends should match the "server" groups from the const hasher

NOTE:::: the ordering is important, first match = first sent

[graphite-regex-map]
default_backend="graphite-proxy"
listen_server="graphite-proxy" # this needs to be an actual SOCKET based server not a backend

    # another backend
    [[graphite-regex-map.map]]
    #what we're regexed from the input
    regex="""^servers..*"""
    backend="graphite-proxy"

    # anything that starts with the prefix (in lue of a more expesive regex)
    [[graphite-regex-map.map]]
    prefix="""servers.main-"""
    backend="graphite-statsd"

    [[graphite-regex-map.map]]
    regex="^servers.*"
    reject=true  # special "reject me" type

[statsd-regex-map]
default_backend="statsd-servers"
listen_server="statsd-servers" # this needs to be an actual SOCKET based server not a backend

    # another backend
    [[statsd-regex-map.map]]
    regex="""^stats.timers..*"""   #what we're regexed from the input
    backend="statsd-servers"

    # anything that starts with the prefix (in lue of a more expesive regex)
    [[statsd-regex-map.map]]
    prefix="stats"
    backend="statsd-statsd"

    [[statsd-regex-map.map]]
    regex="""^stats_count..*"""
    reject=true  # special "reject me" type

..

*/

import (
	"github.com/BurntSushi/toml"
	"github.com/op/go-logging"
	"os"
)

const DEFALT_SECTION_NAME = "prereg"

var log = logging.MustGetLogger("prereg")

type ConfigFilter struct {
	Prefix   string `toml:"prefix"`
	RegEx    string `toml:"regex"`
	IsReject bool   `toml:"reject"`
	Backend  string `toml:"backend"`
}

type ConfigMap struct {
	DefaultBackEnd string `toml:"default_backend"`
	ListenServer   string `toml:"listen_server"`

	FilterList []ConfigFilter `toml:"map"`
}

// list of filters
type ListofConfigMaps map[string]ConfigMap

func (l ListofConfigMaps) ParseConfig() (PreRegMap, error) {
	prs := make(PreRegMap)

	for chunk, cfg := range l {
		pr := new(PreReg)

		pr.Name = chunk
		pr.DefaultBackEnd = cfg.DefaultBackEnd
		pr.ListenServer = cfg.ListenServer

		if len(cfg.DefaultBackEnd) == 0 {
			log.Critical("Need a default_backend for PreReg filters in `%s`", pr.Name)
			os.Exit(1)
		}
		if len(cfg.ListenServer) == 0 {
			log.Critical("Need a listen_server for `%s`", pr.Name)
			os.Exit(1)
		}

		if len(cfg.FilterList) == 0 {
			log.Critical("Need a Some filters for `%s`", pr.Name)
			os.Exit(1)
		}

		//deal with maps
		pr.FilterList = make([]FilterItem, len(cfg.FilterList))
		for idx, cmap := range cfg.FilterList {
			if len(cmap.Prefix) > 0 && len(cmap.RegEx) > 0 {
				log.Critical("Cannot have BOTH `prefix` and `regex` for `%s`", pr.Name)
				os.Exit(1)
			}

			if len(cmap.Prefix) > 0 {
				pf := new(PrefixFilter)
				pf.Prefix = cmap.Prefix
				pf.backend = cmap.Backend
				if len(cmap.Backend) == 0 {
					pf.backend = pr.DefaultBackEnd
				}
				pf.IsReject = cmap.IsReject
				pf.Init()
				pr.FilterList[idx] = pf

			} else {
				pf := new(RegexFilter)
				pf.RegexString = cmap.RegEx
				pf.backend = cmap.Backend
				if len(cmap.Backend) == 0 {
					pf.backend = pr.DefaultBackEnd
				}
				pf.IsReject = cmap.IsReject
				pf.Init()
				pr.FilterList[idx] = pf
			}
		}
		prs[pr.Name] = pr
	}
	prs.LogConfig()
	return prs, nil
}

func ParseConfigFile(filename string) (pr PreRegMap, err error) {

	lcfg := make(ListofConfigMaps)
	if _, err := toml.DecodeFile(filename, &lcfg); err != nil {
		log.Critical("Error decoding config file: %s", err)
		return nil, err
	}

	return lcfg.ParseConfig()
}
