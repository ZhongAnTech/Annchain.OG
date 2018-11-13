import toml
from flask import Flask, request

from common.hosts import hosts

app = Flask(__name__)

# ./og -c http://127.0.0.1:18012/og_config -n run

boot_node_ip = '172.28.152.31'
boot_node_port = 30001
boot_node = '72151a6a292f576bde2f20d29a66c1e0bd1d7f47f182a9da8c7bd8cfe6a349a78bdbccc848057ac27410c12942e3434b531d297007d5fb7cdf10b3a20411fdd7'
parallel = 3
observer_ids = [1, ]


def generate_config_multi_server(id):
    with open('sample.toml') as f:
        d = toml.load(f)

    d['p2p']['bootstrap_nodes'] = "enode://%s@%s:%d" % (boot_node, boot_node_ip, boot_node_port)
    d['leveldb']['path'] = 'data/datadir'
    d['p2p']['bootstrap_node'] = id == 0
    d['auto_client']['sequencer']['enabled'] = d['p2p']['bootstrap_node']

    if id not in observer_ids:
        d['auto_client']['tx']['enabled'] = True
        d['auto_client']['tx']['account_ids'] = [id * parallel + i for i in range(parallel)]    # 3,4,5 for id 1
    else:
        d['auto_client']['tx']['enabled'] = False

    d['debug']['node_id'] = id
    s = toml.dumps(d)
    return s


host_ip_id_map = {}

host_ips = hosts('data/hosts')
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
