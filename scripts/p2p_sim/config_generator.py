import sys

import toml

boot_node = '74e0c7205870b89fda4ed9c10113cd5ac2d274575916a2cdaf7832073e94f7a1dbae433a91bef514c7a0b7653170843eb40cd6ce5c64a373007ce398c9932a81'


def generate_config(id, seq_enabled):
    with open('sample.toml') as f:
        d = toml.load(f)

    port_add = 7300 + id * 10

    d['rpc']['port'] = port_add + 0
    d['p2p']['port'] = port_add + 1
    d['p2p']['bootstrap_node'] = id == 0
    d['p2p']['bootstrap_nodes'] = "enode://%s@127.0.0.1:%d" % (boot_node, 7301)

    d['websocket']['port'] = port_add + 2
    d['profiling']['port'] = port_add + 3
    d['leveldb']['path'] = 'data/datadir_%02d' % (id)

    d['auto_client']['sequencer']['enabled'] = seq_enabled
    d['auto_client']['tx']['enabled'] = False
    d['auto_client']['tx']['account_ids'] = [id,]

    d['debug']['node_id'] = id

    with open('configs/config_%02d.toml' % (id), 'w') as f:
        toml.dump(d, f)

    return d


if __name__ == '__main__':
    total = len(sys.argv) <= 1 and 100 or sys.argv[1]
    enable_sequencer = len(sys.argv) <= 2 or sys.argv[2] == '1'

    print('Total nodes: %d. Sequencer: %s' % (total, enable_sequencer))

    for i in range(total):
        generate_config(i, i == 0 and enable_sequencer)
