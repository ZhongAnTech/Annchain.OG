import datetime
import json

import websocket

try:
    import thread
except ImportError:
    import _thread as thread

start_count_time = datetime.datetime.now()
last_count_time = datetime.datetime.now()
sum_counted = 0
sum_tocount = 0


def hosts(fname):
    with open(fname) as f:
        return [line.strip() for line in f]


def on_message(ws, message):
    global sum_tocount, last_count_time, sum_counted
    j = json.loads(message)
    if j['type'] == 'new_unit':
        seconds = (datetime.datetime.now() - last_count_time).total_seconds()
        if seconds < 1:
            sum_tocount += len(j['nodes'])
        else:
            sum_counted += sum_tocount
            tps = float(sum_tocount) / seconds
            tps_start = float(sum_counted) / (datetime.datetime.now() - start_count_time).total_seconds()
            print("TPS: %0.3f From Start: %0.3f" % (tps, tps_start))
            last_count_time = datetime.datetime.now()
            sum_tocount = 0


def on_error(ws, error):
    print(error)


def on_close(ws):
    print("### closed ###")


def on_open(ws):
    def run(*args):
        ws.send(json.dumps({"event": "new_unit"}))
        ws.send(json.dumps({"event": "confirmed"}))

    thread.start_new_thread(run, ())


if __name__ == "__main__":
    websocket.enableTrace(True)
    host_ips = hosts('hosts')

    ws = websocket.WebSocketApp("ws://%s:30002/ws" % (host_ips[1]),
                                on_message=on_message,
                                on_error=on_error,
                                on_close=on_close)
    ws.on_open = on_open
    ws.run_forever()
