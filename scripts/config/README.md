# Config

## Local
You may start multiple OGs on the same computer.

It will use sample.toml as template (with some modifications on node_id and auto_tx generator) 
```
python3 local_run.py
```

## Config Server
Use a config server for distributed testing.
First pick up a host as config server. 
```
./run_server.sh
```
Server will run on port 30030 and waiting for requests.

To let clients use its configuration judged by IP address, use:
```
# Configuration test:
curl http://172.28.152.31:30030/og_config

# Single instance:
./sync/og -c http://172.28.152.31:30030/og_config -l data/log -n run

# Batch instances:
pssh --user admin --par 20 --hosts data/hosts "chmod +x sync/og && rm -rf data/log && (nohup ./sync/og -c http://172.28.152.31:30030/og_config -l data/log -n run >og.log 2>&1 </dev/null &)"
```

