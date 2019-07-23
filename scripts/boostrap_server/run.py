import os
import threading
import time

from http.server import BaseHTTPRequestHandler, HTTPServer
import json
import cgi
from urllib.parse import parse_qs

ttl = 60 * 2
PARTNERS = 2

# req:
# {"networkid": 1, "publickey":"0x01000000000", "partners": 4, "onode": "onode://d2469187c351fad31b84f4afb2939cb19c03b7c9359f07447aea3a85664cd33d39afc0c531ad4a8e9ff5ed76b58216e19b0ba208b45d5017ca40c9bd351d29ee@47.100.222.11:8001"}

# resp:
# {"status":"wait/ok/error",
# "bootstrap_node": true,
# "bootstrap_nodes": "onode://d2469187c351fad31b84f4afb2939cb19c03b7c9359f07447aea3a85664cd33d39afc0c531ad4a8e9ff5ed76b58216e19b0ba208b45d5017ca40c9bd351d29ee@47.100.222.11:8001",
# "genesis_pk": "genesis_pk"}
networks = {}


def handle(req, sourceip, requirePartners):
    network_id = req['networkid']
    publickey = req['publickey']
    # partners = req['partners']
    onode: str = req['onode']
    # modify onode's ip to source ip
    origin_host = onode[onode.index("@") + 1:onode.index(":", 10)]
    if origin_host == '127.0.0.1':
        onode = onode[0: onode.index("@") + 1] + sourceip + onode[onode.index(":", 10):]
        print('replaced %s to %s' % (origin_host, sourceip))
    req['onode'] = onode

    if network_id not in networks:
        networks[network_id] = {
            'time': time.time(),
            'require': requirePartners,
            'peers': {}
        }
    else:
        if networks[network_id]['require'] != requirePartners:
            msg = {'status': 'error',
                   'message': 'someone else specified required partners as %d, earlier than your %d' % (
                       networks[network_id]['require'], requirePartners)}
            print(msg)
            return msg

    d = networks[network_id]

    if publickey not in d['peers']:
        req['id'] = len(d['peers'])
        d['peers'][publickey] = req

    if len(d['peers']) != d['require']:
        msg = {'status': 'wait',
               'message': '%d of %d is registered... waiting for more' % (len(d['peers']), d['require'])}
        print(msg)
        return msg
    items = d['peers'].values()
    msg = {'status': 'ok',
           'bootstrap_node': d['peers'][publickey]['id'] == 0,
           'bootstrap_nodes': ';'.join([x['onode'] for x in items]),
           'genesis_pk': ';'.join([x['publickey'] for x in items]),
           'partners': len(d['peers'])
           }
    print(msg)
    return msg


class Server(BaseHTTPRequestHandler):
    def __init__(self, *args, directory=None, **kwargs):
        self.lock = threading.Lock()
        t = threading.Thread(target=self.wipe)
        t.daemon = True
        t.start()

        if directory is None:
            directory = os.getcwd()
        self.directory = directory
        super().__init__(*args, **kwargs)

    def _set_headers(self, length):
        self.send_response(200)
        self.send_header('Content-type', 'application/json')
        self.send_header('Content-Length', length)
        self.end_headers()

    def do_HEAD(self):
        self._set_headers(0)

    # GET sends back a Hello world message
    def do_GET(self):
        j = bytes(json.dumps(networks, indent=4), 'utf-8')
        self._set_headers(len(j))
        self.wfile.write(j)

    # POST echoes the message adding a JSON field
    def do_POST(self):
        ctype, pdict = cgi.parse_header(self.headers.get('content-type'))

        # refuse to receive non-json content
        if ctype != 'application/json':
            self.send_response(400)
            self.end_headers()
            return

        # read the message and convert it into a python dictionary
        length = int(self.headers.get('content-length'))
        message = json.loads(self.rfile.read(length))
        required = int(self.path[1:])

        self.lock.acquire()
        try:
            resp = handle(message, self.client_address[0], required)
        finally:
            self.lock.release()

        # send the message back
        j = bytes(json.dumps(resp), 'utf-8')
        self._set_headers(len(j))
        self.wfile.write(j)

    def wipe(self):
        while True:
            self.lock.acquire()
            try:
                to_remove = []
                for k, v in networks.items():
                    if time.time() - v['time'] > ttl:
                        # remove it
                        to_remove.append(k)

                for k in to_remove:
                    print('Removing network id', k)
                    del networks[k]
            finally:
                self.lock.release()
            time.sleep(3)


def run(server_class=HTTPServer, handler_class=Server, port=8008):
    server_address = ('0.0.0.0', port)
    httpd = server_class(server_address, handler_class)

    print('Starting httpd on port %d...' % port)
    httpd.serve_forever()


if __name__ == "__main__":
    from sys import argv

    if len(argv) == 2:
        run(port=int(argv[1]))
    else:
        run()
