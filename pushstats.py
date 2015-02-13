import string
import random
import time
import sys
from multiprocessing import Pool
import socket

MSG_FMT = "{}:{}|c"

def rand_str(chars, llen):
    return ''.join(random.choice(chars) for _ in range(llen))

rand_words = [rand_str(string.ascii_lowercase, 5) for x in range(1,10)]

rand_words = ["test", "house", "here", "there", "cow", "now"]

def gen_key():
    return "{}.{}.{}".format(
        random.choice(rand_words),
        random.choice(rand_words),
        random.choice(rand_words)
    )
    
FULL_CT = 0
PACKET_FULL_CT = 0
#p = Pool(10)
START=time.time()


ON_IP="localhost"
ON_PORT=6000

#ON_IP="192.168.59.103"
#ON_PORT=8126


def send_tcp(msg):
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.connect((ON_IP, ON_PORT,))
    s.send(msg_full)
    s.close()

def send_udp(msg):
    sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM) # UDP
    sock.sendto(msg, (ON_IP, ON_PORT,))


def gen_msg(num):
    global FULL_CT, START,PACKET_FULL_CT
    msg_full = ""
    ct = 0
    
    #now connect to the web server on port 80
    # - the normal http port
    while sys.getsizeof(msg_full) < num:
        msg = MSG_FMT.format(
            gen_key(),
            ct
        )
        msg_full += msg +"\n"
        ct += 1
        FULL_CT +=1
    else:
        send_udp(msg_full)
        PACKET_FULL_CT += 1
        time.sleep(0.001)
        t_delta = time.time()- START
        out_msg = "Sent {}/{} lines ~{:.2f}/s | {} Packets {:.2f}/s".format(
                ct, FULL_CT, float(FULL_CT)/float(t_delta),
                PACKET_FULL_CT, float(PACKET_FULL_CT)/float(t_delta)
        )
        sys.stdout.write(out_msg + ("\b" * len(out_msg)))
        ct = 0

while 1:
    gen_msg(512)
