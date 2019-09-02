import os
import subprocess

ROOT = 'D:/tmp/debug'

if __name__ == '__main__':
    files = os.listdir(ROOT)
    for p in files:
        with open(os.path.join(ROOT, p)) as f:
            s = set(f)
            print(p, len(s))
