import subprocess

import hosts

SSH_PASSWORD_FILE = 'data/passwd'
SSH_HOSTS_FILE = 'data/hosts'
SSH_USERNAME = 'admin'

if __name__ == '__main__':
    hs = hosts.hosts(SSH_HOSTS_FILE)

    pattern = 'sshpass -f %s ssh-copy-id -o StrictHostKeyChecking=no %s@%s'
    for h in hs:
        p = pattern % (SSH_PASSWORD_FILE, SSH_USERNAME, h)
        print(p)
        print(subprocess.run(p, shell=True, check=True))
