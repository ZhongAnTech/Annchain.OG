import datetime
import json
import subprocess
import time

import requests

import hosts

LOCAL_ROOT_FOLDER = '/ws/go/src/github.com/annchain/OG/build/'
REMOTE_ROOT_FOLDER = '/home/admin/sync/'
files = ['og']
USER = 'admin'
ROOT_IP = '172.28.152.12'
MGMT_PORT = 45000

target_hosts = hosts.hosts('data/hosts')
# target_hosts = [
#     '172.28.152.25'
# ]

s = requests.Session()
s.trust_env = False
s.auth = ('gofd', 'gofd')


def wait_done(tid):
    while True:
        resp = s.get('http://%s:%d/api/v1/server/tasks/' % (ROOT_IP, MGMT_PORT) + tid)
        j = json.loads(resp.text)
        if j['status'] == 'COMPLETED':
            print('Completed')
            break
        for host, status in j['dispatchInfos'].items():
            print(host, status['status'], status['percentComplete'])
        print()
        time.sleep(1)


def announce_task():
    j = {"id": datetime.datetime.now().strftime('%Y%m%d%H%M%S'),
         "dispatchFiles": [REMOTE_ROOT_FOLDER + x for x in files],
         "destIPs": target_hosts
         }

    resp = s.post('http://%s:%d/api/v1/server/tasks' % (ROOT_IP, MGMT_PORT), data=json.dumps(j),
                  headers={"Content-type": "application/json"})

    print(resp.status_code, j['id'])
    return j['id']


if __name__ == '__main__':
    for file in files:
        p = "rsync -aP %s %s@%s:%s" % (LOCAL_ROOT_FOLDER + file, USER, ROOT_IP, REMOTE_ROOT_FOLDER)
        print(subprocess.run(p, shell=True, check=True))
    print('Seed ready')

    tid = announce_task()
    wait_done(tid)
