
# test the output of the Graphite consistent hash to see if we can match it with ours

from pprint import pprint

from hasher import ConsistentHashRing

servers = ["s1", "s2", "s3", "s4", "s5", "s6"]
data = """A written look gloves the here lyric. How does a separated helmet chalk? The minister intervenes across the beautiful bliss. The thankful equilibrium gangs your stationary apple.""".split(" ")

ch = ConsistentHashRing(servers)
pprint( ch.ring)
for d in data:
    print d, ch.get_node(d)