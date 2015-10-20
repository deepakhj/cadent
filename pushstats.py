#!/usr/bin/env python

import string
import random
import time
import sys
import socket
from multiprocessing import Process
import argparse

MSG_FMT = "{}:{}|c"

def rand_str(chars, llen):
    return ''.join(random.choice(chars) for _ in range(llen))

rand_words = [rand_str(string.ascii_lowercase, 5) for x in range(1,10)]

rand_words = ["test", "house", "here", "there", "cow", "now"]

def gen_statsd_key(ct):
    return "botests.{}.{}.{}:{}|c".format(
        random.choice(rand_words),
        random.choice(rand_words),
        random.choice(rand_words),
        ct
    )

def gen_graphite_key(ct):
    return "botests-graphite.{}.{}.{} {} {}".format(
        random.choice(rand_words),
        random.choice(rand_words),
        random.choice(rand_words),
        ct,
        int(time.time())
    )

FULL_CT = 0
PACKET_FULL_CT = 0
START=time.time()


ON_SERVER=[
    ("localhost",8125,),
    ("localhost",8125,),
]

#ON_IP="192.168.59.103"
#ON_PORT=8126


def send_tcp(msg, ip, port):
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.connect((ip, port,))
    s.send(msg)
    s.close()

def send_udp(msg, ip, port):
    sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM) # UDP
    sock.sendto(msg, (ip, port,))

def run(ip, port, type, idx, rate):
    msg_func = gen_statsd_key
    send_func = send_udp
    if type == "graphite":
        msg_func = gen_graphite_key
        send_func = send_tcp
    def gen_msg(num):
        global FULL_CT, START,PACKET_FULL_CT
        msg_full = ""
        ct = 0

        #now connect to the web server on port 80
        # - the normal http port
        while sys.getsizeof(msg_full) < num:
            msg = msg_func(ct)
            msg_full += msg +"\n"
            ct += 1
            FULL_CT +=1
        else:
            #print(msg_full)
            send_func(msg_full, ip, port)
            PACKET_FULL_CT += 1
            time.sleep(rate)
            t_delta = time.time()- START
            out_msg = "[{}:{}] Sent {}/{} lines ~{:.2f}/s | {} Packets {:.2f}/s".format(
                ip, port,
                ct, FULL_CT, float(FULL_CT)/float(t_delta),
                PACKET_FULL_CT, float(PACKET_FULL_CT)/float(t_delta)
            )
            sys.stdout.write(out_msg + ("\b" * len(out_msg)))
            ct = 0

    while 1:
        gen_msg(512)

if __name__ == '__main__':

    parser = argparse.ArgumentParser(description='Push Stats')
    parser.add_argument('--type',
                    help='statd or graphite - default %(default)s', default="statsd")
    parser.add_argument('--port', type=int,
                        help='server port # - default %(default)s', default=8125)
    parser.add_argument('--numforks', type=int,
                        help='number of forks to run - default %(default)s', default=2)
    parser.add_argument('--host',
                        help='host name - default %(default)s', default="localhost")
    parser.add_argument('--rate', type=float,
                    help='send rate - default %(default)s', default=0.01)
    parser.add_argument('--depth', type=int,
                    help='number of random of stat permutatins - default %(default)s',
                    default=5)

    args = parser.parse_args()
    idx = 1

    if args.depth > 5:
        rand_words = [rand_str(string.ascii_lowercase, 5) for x in range(1,args.depth)]


    for i in range(0, args.numforks):
        print( "Running to {}:{}".format(args.host, args.host))
        p = Process(target=run, args=(args.host, args.port, args.type, idx, args.rate))
        p.start()
        idx += 1
