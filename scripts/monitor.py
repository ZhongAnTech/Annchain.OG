import datetime
import json
import multiprocessing
import time
import traceback

import pandas as pd
import requests

i = 2

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
        d[host] = doone(host)

    return pd.DataFrame.from_dict(d, orient='index')


if __name__ == '__main__':
    while True:
        df = doround(['127.0.0.1:8000', '127.0.0.1:8100', ])
        print (datetime.datetime.now())
        print(df)
        time.sleep(1)
