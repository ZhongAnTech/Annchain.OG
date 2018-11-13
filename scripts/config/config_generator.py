import sys

import toml


def generate_config_local_server(id, seq_enabled):
    '''
    Generate batch of configurations so that you may start multiple OGs in local computer
    :param id:
    :param seq_enabled: whether this node id supports sequencer auto generation. Should be True for node 0
    :return:
    '''
    with open('sample.toml') as f:
        d = toml.load(f)

    port_add = 8000 + id * 100

    d['rpc']['port'] = port_add + 0
    d['p2p']['port'] = port_add + 1
    d['p2p']['bootstrap_node'] = id == 0
    d['p2p']['bootstrap_nodes'] = "enode://%s@127.0.0.1:%d" % (boot_node, 8001)

    d['websocket']['port'] = port_add + 2
    d['profiling']['port'] = port_add + 3
    d['leveldb']['path'] = 'data/datadir' % (id)

    d['auto_client']['sequencer']['enabled'] = seq_enabled
    d['auto_client']['tx']['enabled'] = True
    d['auto_client']['tx']['account_ids'] = [id, ]

    d['debug']['node_id'] = id

    with open('configs/config_%02d.toml' % (id), 'w') as f:
        toml.dump(d, f)

    return d


if __name__ == '__main__':
    total = len(sys.argv) <= 1 and 10 or sys.argv[1]
    enable_sequencer = len(sys.argv) <= 2 or sys.argv[2] == '1'

    print('Total nodes: %d. Sequencer: %s' % (total, enable_sequencer))

    for i in range(total):
        generate_config_local_server(i, i == 0 and enable_sequencer)
