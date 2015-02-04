import string
import random
import time
import sys
from multiprocessing import Pool
import socket

MSG_FMT = "{} {}"

def rand_str(chars, llen):
    return ''.join(random.choice(chars) for _ in range(llen))

rand_words = [rand_str(string.ascii_lowercase, 5) for x in range(1,10)]

def gen_key():
    return "{}.{}.{}".format(
        random.choice(rand_words),
        random.choice(rand_words),
        random.choice(rand_words)
    )
    
ct = 0
#p = Pool(10)


def gen_msg(num):
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
    else:
        s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        s.connect(("localhost", 6000))
        s.send(msg_full)
        time.sleep(0.001)
        s.close()
        ct = 0

while 1:
    gen_msg(1024)
