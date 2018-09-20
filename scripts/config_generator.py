import sys

import toml

parallel_tx = 1
static_node = 'a4d8435e9923d6cc950ff47df5f8fef469b1acfa255cf9b5daab53abdca3d76403bda33ebedd1336d526ffc510c9420c7df0731ee37ea6b28780ca6011fe741e'


def generate_config(id, seq_enabled):
    with open('sample.toml') as f:
        d = toml.load(f)

    port_add = 8000 + id * 100

    d['rpc']['port'] = port_add + 0
    d['p2p']['port'] = port_add + 1
    d['websocket']['port'] = port_add + 2
    d['profiling']['port'] = port_add + 3
    d['leveldb']['path'] = 'datadir_%02d' % (id)

    d['auto_tx']['enabled'] = True
    # d['auto_tx']['count'] = parallel_tx
    d['auto_sequencer']['enabled'] = seq_enabled
    d['debug']['node_id'] = id

    d['p2p']['static_nodes'] = "enode://%s@127.0.0.1:%d" % (static_node, 8001)

    with open('config_%02d.toml' % (id), 'w') as f:
        toml.dump(d, f)

    return d


if __name__ == '__main__':
    total = int(sys.argv[1])
    enable_sequencer = sys.argv[2] == '1'

    print('Total nodes: %d. Sequencer: %s' % (total, enable_sequencer))

    for i in range(total):
        generate_config(i, i == 0 and enable_sequencer)

