from requests import Session

if __name__ == '__main__':
    s = Session()
    s.trust_env = False
    resp = s.get('http://127.0.0.1:8008/')

    print(resp.text)
    resp = s.get('http://127.0.0.1:8008/')
    print(resp.text)
    resp = s.get('http://127.0.0.1:8008/')
    print(resp.text)

    resp = s.post('http://127.0.0.1:8008/', json={"networkid": 1, "publickey": "0x01000000001", "partners": 4,
                                                  "onode": "onode://d2429ee@47.100.222.11:8001"})
    print(resp.text)
    resp = s.post('http://127.0.0.1:8008/', json={"networkid": 1, "publickey": "0x01000000002", "partners": 4,
                                                  "onode": "onode://d2429ee@47.100.222.11:8001"})
    print(resp.text)
    resp = s.post('http://127.0.0.1:8008/', json={"networkid": 1, "publickey": "0x01000000003", "partners": 4,
                                                  "onode": "onode://d2429ee@47.100.222.11:8001"})
    print(resp.text)
    resp = s.post('http://127.0.0.1:8008/', json={"networkid": 1, "publickey": "0x01000000004", "partners": 4,
                                                  "onode": "onode://d2429ee@47.100.222.11:8001"})
    print(resp.text)
    resp = s.post('http://127.0.0.1:8008/', json={"networkid": 1, "publickey": "0x01000000001", "partners": 4,
                                                  "onode": "onode://d2429ee@47.100.222.11:8001"})
    print(resp.text)
    resp = s.post('http://127.0.0.1:8008/', json={"networkid": 1, "publickey": "0x01000000002", "partners": 4,
                                                  "onode": "onode://d2429ee@47.100.222.11:8001"})
    print(resp.text)
    resp = s.post('http://127.0.0.1:8008/', json={"networkid": 1, "publickey": "0x01000000003", "partners": 4,
                                                  "onode": "onode://d2429ee@47.100.222.11:8001"})
    print(resp.text)
    resp = s.post('http://127.0.0.1:8008/', json={"networkid": 1, "publickey": "0x01000000004", "partners": 4,
                                                  "onode": "onode://d2429ee@47.100.222.11:8001"})
    print(resp.text)
