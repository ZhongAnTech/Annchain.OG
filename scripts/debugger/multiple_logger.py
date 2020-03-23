import asyncio
import datetime
import json
import random
import socketserver
from queue import Queue
from threading import Thread

import websockets
import socket

from websocket_server import WebsocketServer

HOST = '127.0.0.1'  # Standard loopback interface address (localhost)
PORT = 1088  # Port to listen on (non-privileged ports are > 1023)
WS_PORT = 1099

queue = Queue(maxsize=10000)
l = [json.dumps({'level': 'info', 'message': '[0] test0'}),
     json.dumps({'level': 'info', 'message': '[1] test1'}),
     json.dumps({'level': 'info', 'message': '[2] test2'}),
     json.dumps({'level': 'info', 'message': '[3] test3'})]
server = WebsocketServer(WS_PORT)


# Called for every client connecting (after handshake)
def new_client(client, server: WebsocketServer):
    print("New client connected and was given id %d" % client['id'])
    for ll in l:
        # send old messages
        server.send_message(client, ll)


# Called for every client disconnecting
def client_left(client, server):
    print("Client(%d) disconnected" % client['id'])


# Called when a client sends a message
def message_received(client, server, message):
    if len(message) > 200:
        message = message[:200] + '..'
    print("Client(%d) said: %s" % (client['id'], message))
    client['handler'].send_message('I received your ' + message)


def start_websocket_server():
    server.set_fn_new_client(new_client)
    server.set_fn_client_left(client_left)
    server.set_fn_message_received(message_received)
    server.run_forever()


# async def wssend(websocket, path):
#     print('oda')
#     for ll in l:
#         await websocket.send(ll)
#         print('sent')
#
#     while True:
#         data = queue.get()
#         l.append(data)
#         try:
#             await websocket.send(data)
#         except:
#             print('err')
#             return
#     print('over')

class MyTCPHandler(socketserver.StreamRequestHandler):
    def handle(self):
        for line in self.rfile:
            self.data = line.strip()
            if len(self.data) == 0:
                print('returned')
                return
            print(self.data.decode('utf-8'))
            queue.put(self.data.decode('utf-8'))


def start_socket_listener():
    with socketserver.ThreadingTCPServer((HOST, PORT), MyTCPHandler) as server:
        server.serve_forever()


def consume():
    while True:
        q = queue.get()
        # l.append(q)
        server.send_message_to_all(q)


if __name__ == '__main__':
    t = Thread(target=start_socket_listener)
    t.start()
    t = Thread(target=start_websocket_server)
    t.start()
    t = Thread(target=consume)
    t.start()
