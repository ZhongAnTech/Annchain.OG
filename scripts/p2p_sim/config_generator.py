
import sys
import socket
import toml

boot_node = 'a4d8435e9923d6cc950ff47df5f8fef469b1acfa255cf9b5daab53abdca3d76403bda33ebedd1336d526ffc510c9420c7df0731ee37ea6b28780ca6011fe741e'
boot_node_key = '3fa29b2f6b83e037e2573545a6d9c06c0809aeb929cc8c14f992546ae5530b7d'
boot_ip = '172.28.152.31'
#boot_ip = '127.0.0.1'

def generate_config(id, seq_enabled):
    with open('sample.toml') as f:
        d = toml.load(f)

    port_add = 11300 + id * 10

    d['rpc']['port'] = port_add + 0
    d['p2p']['port'] = port_add + 1
    d['p2p']['bootstrap_node'] = seq_enabled
    d['p2p']['bootstrap_nodes'] = "onode://%s@%s:%d" % (boot_node,boot_ip, 11301)
    if (seq_enabled ):
        d['p2p']['node_key'] = boot_node_key
    d['websocket']['port'] = port_add + 2
    d['profiling']['port'] = port_add + 3
    d['leveldb']['path'] = 'data/datadir_%02d' % (id)
    #change it to true manualy
    d['auto_client']['sequencer']['enabled'] = seq_enabled
    d['auto_client']['tx']['enabled'] = False
    d['auto_client']['tx']['account_ids'] = [id,]

    d['debug']['node_id'] = id

    with open('configs/config_%02d.toml' % (id), 'w') as f:
        toml.dump(d, f)

    return d


def public_ip():
    try:
        s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        s.connect(('114.114.114.114', 80))
        ip = s.getsockname()[0]
    finally:
        s.close()
    return ip


if __name__ == '__main__':
    total = len(sys.argv) <= 1 and 100 or sys.argv[1]
    ip = public_ip()
    enable_sequencer = False
    if  ip == boot_ip or boot_ip == '127.0.0.1' :
        enable_sequencer = True
    print('Total nodes: %d. Sequencer: %s' % (total, enable_sequencer))

    for i in range(total):
        generate_config(i, i == 0 and enable_sequencer)
