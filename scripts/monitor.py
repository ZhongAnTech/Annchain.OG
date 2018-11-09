import datetime
import json
import multiprocessing
import time

import pandas as pd
import requests

# pd.set_option('display.height', 1000)
pd.set_option('display.max_rows', 500)
pd.set_option('display.max_columns', 500)
pd.set_option('display.width', 1000)

total = 20

s = requests.Session()
s.trust_env = False


id_host_map = {}
host_id_map = {}


def host_to_show(ip):
    return ip.split(':')[0].split('.')[3]


def id_to_show(id):
    if id not in id_host_map:
        return id + ':N/A'
    return host_to_show(id_host_map[id])


def myid(host):
    try:
        resp = s.get('http://%s/net_info' % host, timeout=10)
        j = json.loads(resp.text)
        return j['short_id']
    except Exception as e:
        raise e


def doone(host):
    try:
        d = {}
        try:
            resp = s.get('http://%s/sequencer' % host, timeout=5)
            j = json.loads(resp.text)
            d['seq'] = j['Id']
        except Exception as e:
            return None
            # d['seq'] = -1

        try:
            peers = []
            resp = s.get('http://%s/peers_info' % host, timeout=5)
            j = json.loads(resp.text)
            for peer in j:
                peers.append(peer['short_id'])
                d['peers'] = peers
        except Exception as e:
            return None

        try:
            resp = s.get('http://%s/sync_status' % host, timeout=5)
            j = json.loads(resp.text)
            d.update(j)
            d['error'] = d['error']
            d['syncMode'] = d['syncMode'].strip()[10:][0:4]
            d['catchupSyncerStatus'] = d['catchupSyncerStatus'].strip()[3:]
        except Exception as e:
            return None

        return host, d
    except KeyboardInterrupt as e:
        return
    except Exception as e:
        return None

def doround(hosts, pool):
    r = {}
    ever = False
    for host in hosts:
        if host not in host_id_map:
            id = myid(host)
            if id is not None:
                host_id_map[host] = id
                id_host_map[id] = host

    for c in pool.imap(doone, hosts):
        if c is None:
            continue
        host, d = c
        if d is not None:
            d['bestPeer'] = id_to_show(d['bestPeer'])
            ippeers = []
            for peer in d['peers']:
                ippeers.append(id_to_show(peer))
            d['peers'] = ippeers
            d['peers'] = len(ippeers)
            d['id'] = d['id'][0:4]

            r[host_to_show(host)] = d
            ever = True
    if not ever:
        return None

    # return pd.DataFrame.from_dict(d, orient='index')
    return pd.DataFrame.from_dict(r)


def hosts(fname):
    with open(fname) as f:
        return [line.strip() for line in f]


if __name__ == '__main__':
    # hosts = ['127.0.0.1:%d' % (8000 + i*100) for i in range(total)]
    host_ips = hosts('hosts')
    host_ipports = ['%s:30000' % (x) for x in host_ips]
    pool = multiprocessing.Pool(processes=20)

    try:
        while True:
            try:
                df = doround(host_ipports, pool)
                if df is not None:
                    print("=" * 20)
                    print(datetime.datetime.now())
                    print(df)
                time.sleep(1)
            except KeyboardInterrupt as e:
                raise e
            except:
                pass
            finally:
                time.sleep(1)
    except KeyboardInterrupt as e:
        print('Ending')
        pool.terminate()
    finally:
        pool.join()
