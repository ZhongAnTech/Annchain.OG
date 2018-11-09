# Deploy Script
Deploy scripts help deploy binaries to multiple hosts instantly, by leveraing P2P content distribution.

## Prerequisites
sudo apt-get install sshpass

install pssh and then pkill pnuke will all be there

## Actions
# add ssh keys
batch_sshcopyid.py

# batch commands
pssh --user admin --par 20 -i --hosts data/hosts echo "xxx"
pssh --user admin --par 20 -i --hosts data/hosts rm cmd download og release
pssh --user admin --par 20 -i --hosts data/hosts /ws/go/src/github.com/latifrons/gofd/client.sh
prsync -av --user admin --hosts data/hosts -a /ws/go/src/github.com/latifrons/gofd/release/ /home/admin/gofd

pssh --user admin --par 20 -i --hosts data/hosts "cd /home/admin/gofd; ./agent.sh"
pssh --user admin --par 20 -i --hosts data/hosts "ps aux | grep gofd"

pnuke --user admin --par 20 --hosts data/hosts gofd

# start og
./sync/og -c http://172.28.152.31:30030/og_config -n run
./sync/og -c http://47.101.139.203:30030/og_config -n run

# batch start og
pssh --user admin --par 20 --hosts data/hosts "chmod +x sync/og && rm -rf data/log && nohup ./sync/og -c http://172.28.152.31:30030/og_config -l data/log -n run >og.log 2>&1 </dev/null &"

# batch stop og
pnuke --user admin --par 20 --hosts data/hosts og

# check alive
pssh --user admin --par 20 -i --hosts data/hosts "ps aux | grep sync/og | wc -l "