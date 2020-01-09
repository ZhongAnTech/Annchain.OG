import io
import os
import yaml

# hosts = {
#     'izuf63aqm171s25itl5egkz': {'v1': 200, 'v2': 100, 'v3': 100, 'v4': 100},
#     'izuf63aqm171s25itl5egjz': {'v1': 40, 'v2': 20, 'v3': 20},
#     'izuf6a8si7x4swoav7odv5z': {'v1': 40, 'v2': 20, 'v3': 20},
#     'izuf6a8si7x4swoav7odv6z': {'v1': 40, 'v2': 20, 'v3': 20},
#     'izuf6a8si7x4swoav7odv7z': {'v1': 40, 'v2': 20, 'v3': 20},
#     'izuf6a8si7x4swoav7odv8z': {'v1': 40, 'v2': 20, 'v3': 20},
# }
#
# host_ip = {
#     'izuf63aqm171s25itl5egkz': '172.28.152.101',
#     'izuf63aqm171s25itl5egjz': '172.28.152.102',
#     'izuf6a8si7x4swoav7odv5z': '172.28.152.104',# 103
#     'izuf6a8si7x4swoav7odv6z': '172.28.152.105',# 104
#     'izuf6a8si7x4swoav7odv7z': '172.28.152.103',# 105
#     'izuf6a8si7x4swoav7odv8z': '172.28.152.106',
# }

storage_class = 'huge'
# storage_class = 'local-storage'
prefix = 'huge'

hosts = {
    'izuf63aqm171s25itl5egkz': {prefix + '1': 50, prefix + '2': 50},
    # 'izuf63aqm171s25itl5egjz': {prefix + '1': 5, prefix + '2': 5},
    # 'izuf6a8si7x4swoav7odv5z': {prefix + '1': 5, prefix + '2': 5},
    # 'izuf6a8si7x4swoav7odv6z': {prefix + '1': 5, prefix + '2': 5},
    # 'izuf6a8si7x4swoav7odv7z': {prefix + '1': 5, prefix + '2': 5},
    # 'izuf6a8si7x4swoav7odv8z': {prefix + '1': 5, prefix + '2': 5},
}

host_ip = {
    'izuf63aqm171s25itl5egkz': '172.28.152.101',
    'izuf63aqm171s25itl5egjz': '172.28.152.102',
    'izuf6a8si7x4swoav7odv5z': '172.28.152.104',  # 103
    'izuf6a8si7x4swoav7odv6z': '172.28.152.105',  # 104
    'izuf6a8si7x4swoav7odv7z': '172.28.152.103',  # 105
    'izuf6a8si7x4swoav7odv8z': '172.28.152.106',
}


def reg(host, folder, size, storage_class):
    with open("sample.yaml", 'r') as stream:
        try:
            d = yaml.safe_load(stream)
            d['metadata']['name'] = host_ip[host][-3:] + '-' + folder
            d['spec']['capacity']['storage'] = '%dGi' % size
            d['spec']['local']['path'] = '/alidata1/nfs/' + folder
            d['spec']['nodeAffinity']['required']['nodeSelectorTerms'][0]['matchExpressions'][0]['values'][0] = host

            if storage_class is None:
                del d['spec']['storageClassName']
            else:
                d['spec']['storageClassName'] = storage_class

            with io.open('output.yaml', 'w', encoding='utf8') as outfile:
                yaml.dump(d, outfile, default_flow_style=False, allow_unicode=True)
                os.system('kubectl create -f output.yaml')
        except yaml.YAMLError as exc:
            print(exc)


if __name__ == '__main__':
    # remove all volumes
    # os.system('kubectl delete pv --all')

    for host, value in hosts.items():
        for folder, size in value.items():
            reg(host, folder, size, storage_class)
