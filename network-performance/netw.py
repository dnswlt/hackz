import socket
import sys
import time


NUM_BYTES = 1000
CHUNK_SIZE = 4096

def run_client(ip_addr, port):
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
        sock.connect((ip_addr, port))
        print("Connected to %s:%d" % (ip_addr, port))
        b_read = 0
        t_before = time.time()
        while b_read < NUM_BYTES:
            data = sock.recv(min(NUM_BYTES, CHUNK_SIZE))
            if len(data) == 0:
                raise IOError("Connection broken")
            b_read += len(data)
        duration = time.time() - t_before
        print(f"Received {NUM_BYTES} bytes in {duration:.3f} seconds.")

    
def run_server(ip_addr, port):
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.bind((ip_addr, port))
    s.listen()
    while True:
        (sock, address) = s.accept()
        b_sent = 0
        data = os.urandom(NUM_BYTES)
        while b_sent < NUM_BYTES:
            sock.send(data[b_sent:min(b_sent+CHUNK_SIZE, NUM_BYTES)])
        sock.close()
    
if __name__ == "__main__":
    if len(sys.argv) == 1:
        print("Usage: 'server <listen-ip> <port>' or 'client <server-ip> <server-port>'")
        sys.exit(1)
    if sys.argv[1] == 'server':
        run_server(sys.argv[2], int(sys.argv[3]))
    elif sys.argv[1] == 'client':
        run_client(sys.argv[2], int(sys.argv[3]))
        