import datetime
import json
import multiprocessing
import time
import traceback

import pandas as pd
import requests

#pd.set_option('display.height', 1000)
pd.set_option('display.max_rows', 500)
pd.set_option('display.max_columns', 500)
pd.set_option('display.width', 1000)


total = 10

s = requests.Session()
s.trust_env = False

id_host_map = {}
host_id_map = {}


def myid(host):
    try:
        resp = s.get('http://%s/net_info' % host, timeout=3)
        j = json.loads(resp.text)
        return j['id']
    except Exception as e:
        return None


def doone(host):
    d = {}
    try:
        resp = s.get('http://%s/sequencer' % host, timeout=3)
        j = json.loads(resp.text)
        d['seq'] = j['Id']
    except Exception as e:
        d['seq'] = -1

    try:
        peers = []
        resp = s.get('http://%s/peers_info' % host, timeout=3)
        j = json.loads(resp.text)
        for peer in j:
            if peer['id'] in id_host_map:
                peers.append(id_host_map[peer['id']][-4:])
        d['peers'] = peers
    except Exception as e:
        d['peers'] = []

    try:
        resp = s.get('http://%s/sync_status' % host, timeout=3)
        j = json.loads(resp.text)
        d.update(j)
        d['syncMode'] = d['syncMode'][10:]
        d['catchupSyncerStatus'] = d['catchupSyncerStatus'][3:]
    except Exception as e:
        pass

    return d


def doround(hosts):
    d = {}
    for host in hosts:
        if host not in host_id_map:
            id = myid(host)
            if id is not None:
                host_id_map[host] = id
                id_host_map[id] = host

    for host in hosts:
        d[host[10:]] = doone(host)

    # return pd.DataFrame.from_dict(d, orient='index')
    return pd.DataFrame.from_dict(d)


if __name__ == '__main__':
    hosts = ['127.0.0.1:%d' % (8000 + i*100) for i in range(total)]
    while True:
        df = doround(hosts)
        print(datetime.datetime.now())
        print(df)
        time.sleep(1)
