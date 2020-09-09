import random
import subprocess
import time
from threading import Thread

stop = False


def run_one(node_id: int):
    f = open("%s.log" % (node_id), 'a')
    fe = open("%s.err.log" % (node_id), 'a')

    cmd = r'D:\ws\og\OG-refactor\arefactor\main.exe run -r D:\ws\og\OG-refactor\arefactor\data\nodedata%d --gen-key --id=%d' % (
        node_id, node_id)
    cmds = cmd.split()
    p = subprocess.Popen(cmds, stdout=f, stderr=fe)

    sleep_time = random.randint(0, 60 * 10)
    print('node running %d %d' % (node_id, sleep_time))
    time.sleep(sleep_time)
    p.kill()
    sleep_time = random.randint(0, 60 * 3)
    print('node killed %d %d' % (node_id, sleep_time))
    time.sleep(sleep_time)

    f.close()
    fe.close()


def keep_run(node_id: int):
    while not stop:
        run_one(node_id)


if __name__ == '__main__':
    ts = []
    for i in range(5):
        t = Thread(target=keep_run, args=[i])
        t.start()

    for t in ts:
        t.join()
