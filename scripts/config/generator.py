import yaml

var = 7

if __name__ == '__main__':
    # ex
    for i in range(var):
        with open('service_ex.yaml') as f:
            y = yaml.load(f)

        y['metadata']['name'] = 'ognb-ex-%d' % (i)
        for portset in y['spec']['ports']:
            portset['nodePort'] = portset['nodePort'] + i * 10
        y['spec']['selector']['statefulset.kubernetes.io/pod-name'] = 'ognb-%d' % (i)

        with open('ex/%d.yaml' % (i), 'w') as f:
            yaml.dump(y, f)

if __name__ == '__main__':
    # in
    for i in range(var):
        with open('service_in.yaml') as f:
            y = yaml.load(f)

        y['metadata']['name'] = 'ognb-%d' % (i)
        y['spec']['selector']['statefulset.kubernetes.io/pod-name'] = 'ognb-%d' % (i)

        with open('in/%d.yaml' % (i), 'w') as f:
            yaml.dump(y, f)