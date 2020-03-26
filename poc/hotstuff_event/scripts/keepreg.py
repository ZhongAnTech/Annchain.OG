import datetime
import os
import signal
import subprocess
import sys
import time
import traceback

import requests
import shlex
from subprocess import Popen, PIPE, STDOUT

HOST = 'http://127.0.0.1:8000'
CMD = 'notepad.exe'

last_set = set()
process = None
ifwin = os.name == 'nt'

if __name__ == '__main__':
    s = requests.Session()

    while True:
        try:
            print(str(datetime.datetime.now()) + "   Registering...")
            resp = s.get(HOST + '/register')
            jj = resp.json()
            print(jj)
            current_set = set(jj['Peers'])
            if last_set == current_set:
                print('no change')
                continue

            # changed. start new process to do the consensus
            with open('peers.lst', 'w') as f:
                f.writelines([x for x in current_set])
            last_set = current_set
            if process is not None:
                print('terminating process: ' + str(process.pid))
                if ifwin:
                    dev_null = open(os.devnull, 'w')
                    command = ['TASKKILL', '/F', '/T', '/PID', str(process.pid)]
                    subprocess.Popen(command, stdin=dev_null, stdout=sys.stdout, stderr=sys.stderr)
                else:
                    process.kill()
                while not process.poll():
                    print('waiting to kill ' + str(process.pid))
                    time.sleep(1)

            process = Popen(shlex.split(CMD), stdout=PIPE, stderr=STDOUT)
            print('process started: ' + str(process.pid))
            # process.communicate()
            # exit_code = process.wait()

        except Exception as e:
            traceback.print_exc()
        finally:
            time.sleep(5)
