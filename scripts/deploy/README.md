# Deploy Script
Deploy scripts help deploy binaries to multiple hosts instantly, by leveraing P2P content distribution.

## Prerequisites
sudo apt-get install sshpass

install pssh and then pkill pnuke will all be there

## Actions
# Add ssh keys 
so that you don't need password to login to the hosts
```
python3 batch_sshcopyid.py
```

# Broadcast files
You need to modifiy parameters in broadcast_file.py first.
```
python3 broadcast_file.py
```

# Batch commands examples
```
# do a global echo
pssh --user admin --par 20 -i --hosts data/hosts echo "xxx"

# rsync files to the target hosts
prsync -av --user admin --hosts data/hosts -a /ws/go/src/github.com/latifrons/gofd/release/ /home/admin/gofd

# start a p2p client to receive files 
pssh --user admin --par 20 -i --hosts data/hosts "cd /home/admin/gofd; ./agent.sh"

# check process status
pssh --user admin --par 20 -i --hosts data/hosts "ps aux | grep gofd"

# kill processes
pnuke --user admin --par 20 --hosts data/hosts gofd
```

## start og using a configuration server
```
# private IP
./sync/og -c http://172.28.152.31:30030/og_config -n run
# public IP
./sync/og -c http://47.101.139.203:30030/og_config -n run
```

## batch start og
```
pssh --user admin --par 20 --hosts data/hosts "chmod +x sync/og && rm -rf data/log && (nohup ./sync/og -c http://172.28.152.31:30030/og_config -l data/log -n run >og.log 2>&1 </dev/null &)"
```

## batch stop og
```
pnuke --user admin --par 20 --hosts data/hosts og
```

## check alive
```
pssh --user admin --par 20 -i --hosts data/hosts "ps aux | grep sync/og | wc -l "
```

## remove all running data (start from beginning)
```
pssh --user admin --par 20 --hosts data/hosts "rm -rf data/log data/datadir og.log"
```
