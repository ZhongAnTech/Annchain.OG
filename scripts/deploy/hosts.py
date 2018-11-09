def hosts(fname):
    with open(fname) as f:
        return [line.strip() for line in f]
