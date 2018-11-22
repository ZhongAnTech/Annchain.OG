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


total = 100

s = requests.Session()
s.trust_env = False

id_host_map = {}
host_id_map = {}

def doone(host):
    d = {}
    try:
        resp = s.get('http://%s/monitor' % host, timeout=3)
        j = json.loads(resp.text)
        d = j
    except Exception as e:
        return None

    return d


def doround(hosts):
    d = {}
    ever = False

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
    hosts = ['127.0.0.1:%d' % (7300 + i*10) for i in range(total)]
    while True:
        df = doround(hosts)
        if df is not None:
            print("=" * 20)
            print(datetime.datetime.now())
            print(df)
        time.sleep(3)
