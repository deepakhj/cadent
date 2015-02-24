import string
import random
import time
import sys
import socket
from multiprocessing import Process

MSG_FMT = "{}:{}|c"

def rand_str(chars, llen):
    return ''.join(random.choice(chars) for _ in range(llen))

rand_words = [rand_str(string.ascii_lowercase, 5) for x in range(1,10)]

rand_words = ["test", "house", "here", "there", "cow", "now"]

def gen_key():
    return "botests.{}.{}.{}".format(
        random.choice(rand_words),
        random.choice(rand_words),
        random.choice(rand_words)
    )

FULL_CT = 0
PACKET_FULL_CT = 0
START=time.time()


ON_SERVER=[
    ("localhost",8125,),
    ("localhost",8125,),
    ("localhost",8125,),
 #   ("localhost", 6001,)
]

#ON_IP="192.168.59.103"
#ON_PORT=8126


def send_tcp(msg, ip, port):
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.connect((ip, port,))
    s.send(msg_full)
    s.close()

def send_udp(msg, ip, port):
    sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM) # UDP
    sock.sendto(msg, (ip, port,))

def run(ip, port, idx):
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
            send_udp(msg_full, ip, port)
            PACKET_FULL_CT += 1
            time.sleep(0.01)
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
    idx = 1
    for ip, port in ON_SERVER:
        print( "Running to {}:{}".format(ip, port))
        p = Process(target=run, args=(ip, port, idx,))
        p.start()
        idx += 1
        #p.join()