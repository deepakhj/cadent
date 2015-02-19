
// note ... need to do this in this dir `npm install hashring --save`
var hashring = require('hashring')
var util = require('util')


var node_status = [];
var node_ring = {};

var ring = new hashring(
    node_ring, 'md5', {
        'max cache size': 10000,
        //We don't want duplicate keys sent so replicas set to 0
        'replicas': 1
    });

var servers = ["s1", "s2", "s3", "s4", "s5", "s6"]
var data = "A written look gloves the here lyric. How does a separated helmet chalk? The minister intervenes across the beautiful bliss. The thankful equilibrium gangs your stationary apple.".split(" ")

servers.forEach(function(element, index, array) {
    node_ring[element] = 100;
    t_node = {}
    t_node[element] = 100
    ring.add(t_node)
});
util.log(ring.get(data[0]))

util.log(ring.ring)


