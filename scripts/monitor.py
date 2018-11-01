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
        return j['short_id']
    except Exception as e:
        return None


def doone(host):
    d = {}
    try:
        resp = s.get('http://%s/sequencer' % host, timeout=3)
        j = json.loads(resp.text)
        d['seq'] = j['Id']
    except Exception as e:
        return None
        # d['seq'] = -1

    try:
        peers = []
        resp = s.get('http://%s/peers_info' % host, timeout=3)
        j = json.loads(resp.text)
        for peer in j:
            if peer['short_id'] in id_host_map:
                peers.append(id_host_map[peer['short_id']][-4:])
        d['peers'] = peers
    except Exception as e:
        d['peers'] = []

    try:
        resp = s.get('http://%s/sync_status' % host, timeout=3)
        j = json.loads(resp.text)
        d.update(j)
        d['bestPeer'] = id_host_map[d['bestPeer']][-4:]
        d['error'] = d['error']
        d['syncMode'] = d['syncMode'][10:]
        d['catchupSyncerStatus'] = d['catchupSyncerStatus'][3:]
    except Exception as e:
        pass

    return d


def doround(hosts):
    d = {}
    ever = False
    for host in hosts:
        if host not in host_id_map:
            id = myid(host)
            if id is not None:
                host_id_map[host] = id
                id_host_map[id] = host

    for host in hosts:
        v = doone(host)
        if v is not None:
            d[host[10:]] = v
            ever = True

    if not ever:
        return None

    # return pd.DataFrame.from_dict(d, orient='index')
    return pd.DataFrame.from_dict(d)


if __name__ == '__main__':
    hosts = ['127.0.0.1:%d' % (8000 + i*100) for i in range(total)]
    while True:
        df = doround(hosts)
        if df is not None:
            print(datetime.datetime.now())
            print(df)
        time.sleep(1)
