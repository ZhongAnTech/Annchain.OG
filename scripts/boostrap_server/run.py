import threading
import time

from http.server import BaseHTTPRequestHandler, HTTPServer
import json
import cgi

ttl = 60 * 5

# req:
# {"networkid": 1, "publickey":"0x01000000000", "partners": 4, "onode": "onode://d2469187c351fad31b84f4afb2939cb19c03b7c9359f07447aea3a85664cd33d39afc0c531ad4a8e9ff5ed76b58216e19b0ba208b45d5017ca40c9bd351d29ee@47.100.222.11:8001"}

# resp:
# {"status":"wait/ok/error",
# "bootstrap_node": true,
# "bootstrap_nodes": "onode://d2469187c351fad31b84f4afb2939cb19c03b7c9359f07447aea3a85664cd33d39afc0c531ad4a8e9ff5ed76b58216e19b0ba208b45d5017ca40c9bd351d29ee@47.100.222.11:8001",
# "genesis_pk": "genesis_pk"}
networks = {}


def handle(req, sourceip):
    network_id = req['networkid']
    publickey = req['publickey']
    partners = req['partners']
    onode: str = req['onode']
    # modify onode's ip to source ip
    onode = onode[0: onode.index("@") + 1] + sourceip + onode[onode.index(":", 10):]
    req['onode'] = onode

    if network_id not in networks:
        networks[network_id] = {
            'time': time.time(),
            'require': partners,
            'peers': {}
        }

    d = networks[network_id]
    if d['require'] != partners:
        return {'status': 'error',
                'message': 'Another one is requiring %d partners previously. You are requiring %d.' % (
                    d['require'], partners)}

    if publickey not in d['peers']:
        req['id'] = len(d['peers'])
        d['peers'][publickey] = req

    if len(d['peer']) != d['require']:
        return {'status': 'wait',
                'message': '%d of %d is registered... waiting for more' % (len(d['peer']), d['require'])}
    return {'status': 'ok',
            'bootstrap_node': req['id'] == 0,
            'bootstrap_nodes': [';'.join([x['onode'] for x in d['peer']])],
            'genesis_pk': [';'.join([x['publickey'] for x in d['peer']])]
            }


#


class Server(BaseHTTPRequestHandler):
    def __init(self):
        self.lock = threading.Lock()

    def _set_headers(self):
        self.send_response(200)
        self.send_header('Content-type', 'application/json')
        self.end_headers()

    def do_HEAD(self):
        self._set_headers()

    # GET sends back a Hello world message
    def do_GET(self):
        self._set_headers()
        self.wfile.write(json.dumps({'hello': 'world', 'received': 'ok'}))

    # POST echoes the message adding a JSON field
    def do_POST(self):
        ctype, pdict = cgi.parse_header(self.headers.getheader('content-type'))

        # refuse to receive non-json content
        if ctype != 'application/json':
            self.send_response(400)
            self.end_headers()
            return

        # read the message and convert it into a python dictionary
        length = int(self.headers.getheader('content-length'))
        message = json.loads(self.rfile.read(length))

        self.lock.acquire()
        try:
            print(self)
            resp = handle(message)
        finally:
            self.lock.release()

        # send the message back
        self._set_headers()
        self.wfile.write(json.dumps(resp))


def run(server_class=HTTPServer, handler_class=Server, port=8008):
    server_address = ('', port)
    httpd = server_class(server_address, handler_class)

    print('Starting httpd on port %d...' % port)
    httpd.serve_forever()


if __name__ == "__main__":
    from sys import argv

    if len(argv) == 2:
        run(port=int(argv[1]))
    else:
        run()
