import datetime
import json
import multiprocessing
import time
import traceback

import pandas as pd
import requests

total = 10

s = requests.Session()
s.trust_env = False


def doone(host):
    d = {}
    try:
        resp = s.get('http://%s/sequencer' % host, timeout=3)
        j = json.loads(resp.text)
        d['seq'] = j['Id']
    except Exception as e:
        d['seq'] = -1
    return d


def doround(hosts):
    d = {}
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
