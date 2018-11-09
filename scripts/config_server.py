import toml
from flask import Flask, request

app = Flask(__name__)

# ./og -c http://127.0.0.1:18012/og_config -n run

boot_node_ip = '172.28.152.31'
boot_node = '74e0c7205870b89fda4ed9c10113cd5ac2d274575916a2cdaf7832073e94f7a1dbae433a91bef514c7a0b7653170843eb40cd6ce5c64a373007ce398c9932a81'


def generate_config_multi_server(id):
    with open('sample.toml') as f:
        d = toml.load(f)

    d['p2p']['bootstrap_nodes'] = "enode://%s@%s:%d" % (boot_node, boot_node_ip, 8001)
    d['leveldb']['path'] = 'data/datadir'
    d['auto_client']['sequencer']['enabled'] = id == 0
    d['auto_client']['tx']['enabled'] = True
    d['auto_client']['tx']['account_ids'] = [id, ]
    d['debug']['node_id'] = id
    s = toml.dumps(d)
    return s


def hosts(fname):
    with open(fname) as f:
        return [line.strip() for line in f]


host_ip_id_map = {}

host_ips = hosts('hosts')
i = 0
for host_ip in host_ips:
    host_ip_id_map[host_ip] = i
    i += 1


@app.route("/og_config")
def hello():
    ip = request.remote_addr
    if ip not in host_ip_id_map:
        return "IP %s is not allowed" % (ip), 403

    return generate_config_multi_server(host_ip_id_map[ip]), 200
