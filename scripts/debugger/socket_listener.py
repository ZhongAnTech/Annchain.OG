import socket

HOST = '127.0.0.1'  # Standard loopback interface address (localhost)
PORT = 1088        # Port to listen on (non-privileged ports are > 1023)

if __name__ == '__main__':
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
        s.bind((HOST, PORT))
        s.listen()
        conn, addr = s.accept()
        with conn:
            print('Connected by', addr)
            while True:
                data = conn.recv(1024)
                if not data:
                    break
                print(data.decode('utf-8'))