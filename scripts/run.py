import select
import subprocess
import sys
from typing import List, Any

import config_generator

binary = '../build/og'
params = '-c config_XX.toml -d datadir_XX -l datadir_XX -n run'

current_node_id = 0
pids: List[subprocess.Popen] = []


def add_node():
    global current_node_id
    global pids

    i = current_node_id
    config_generator.generate_config(i, i == 0)

    p = params.replace('XX', '%02d' % (i))
    print(p)
    pp = p.split(' ')
    pp.insert(0, binary)
    pid = subprocess.Popen(pp, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
    pids.append(pid)
    print('Started %d' % current_node_id)
    current_node_id += 1


def del_node():
    global current_node_id
    global pids

    current_node_id -= 1
    pids[current_node_id].terminate()
    pids[current_node_id].wait()
    pids = pids[:-1]
    print('Terminated %d' % current_node_id)


if __name__ == '__main__':
    for i in range(2):
        add_node()

    while True:
        inp, outp, err = select.select([sys.stdin], [], [])
        c = sys.stdin.read(1)
        if c == 'q':
            while current_node_id > 0:
                del_node()
        if c == '=':
            add_node()
        if c == '-':
            del_node()
