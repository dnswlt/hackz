"""Network speed measurement utility.

This script allows to measure the network throughput between
two hosts. One plays the role of the server (i.e., it accepts
connections from one client at a time, but keeps running);
the other plays the role of the client that connects to the 
server and requests one of the following measurement protocols:

(1) Speed measurement 

[4 bytes][8 bytes           ][8 bytes             ][8 bytes         ]
[b"SPDM"][int64 num_bytes up][int64 num_bytes down][int64 chunk_size]

(2) Shutdown server
[4 bytes]
[b"SHUT"]

(3) Latency measurement
[4 bytes][8 bytes           ][8 bytes             ][8 bytes          ]
[b"LATN"][int64 num pings up][int64 num pings down][int64 packet size]
"""

import argparse
from math import ceil
import os
import socket
import struct
import sys
import time


def parse_args():
    p = argparse.ArgumentParser(description="Network speed measurement utility.")
    p.add_argument("-s", "--host", default="localhost", help="Hostname or IP to connect to (in client mode) or to listen on (in server mode).")
    p.add_argument("-p", "--port", default=10101, type=int)
    p.add_argument("-m", "--mode", default="client", choices=["client", "server"], 
        help="Mode to run in. Start one side as the server and then run tests from the other side as a client.")
    p.add_argument("-c", "--command", default="throughput", choices=["throughput", "latency", "shutdown"],
        help="Specifies the test to run. 'shutdown' kills the server.")
    p.add_argument("-b", "--chunk_size", default=4096, type=int,
        help="Number of bytes to send per packet.")
    p.add_argument("-n", "--num_bytes", default=4096, type=int,
        help="Total number of bytes to send in a 'throughput' test.")
    p.add_argument("-k", "--num_packets", default=500, type=int,
        help="Total number of packets to send in a 'latency' test.")
    return p.parse_args()


def fmt_bytes(n_bytes):
    if n_bytes < 2**10:
        return f"{n_bytes} bytes"
    if n_bytes <= 2**20:
        return f"{n_bytes/2**10:.3f} kiB"
    return f"{n_bytes/2**20:.3f} MiB"


def fmt_thrpt(bps):
    return f"{fmt_bytes(bps)}/s"


def recv_int64(sock):
    buf = bytearray(8)
    n_read = 0
    while n_read < 8:
        n_read += sock.recv_into(buf, 8 - n_read)
    return struct.unpack('!q', buf)[0]


def send_int64(sock, n):
    data = struct.pack('!q', n)
    n_sent = 0
    while n_sent < 8:
        n_sent += sock.send(data[n_sent:])


def stats(ts):
    """Returns a dict with summary stats for given list of (time) measurements."""
    ts = sorted(ts)
    return {
        'count': len(ts),
        'avg': sum(ts) / len(ts),
        'min': ts[0],
        'max': ts[-1],
        'median': ts[len(ts)//2],
        'p95': ts[ceil(len(ts) * 0.95)] if len(ts) >= 100 else ts[-1],
        'p99': ts[ceil(len(ts) * 0.99)] if len(ts) >= 100 else ts[-1],
    }


def fmt_stats(stats):
    return (('  count: %d\n' % stats['count']) +
        '\n'.join('  %s: %.1fms' % (k, v * 1000) for k, v in stats.items() if k != 'count'))


def send_data(sock, num_bytes, chunk_size):
    sent_total = 0
    data = b'\x00' * chunk_size
    t_start = time.time()
    while sent_total < num_bytes:
        n_bytes = min(num_bytes - sent_total, chunk_size)
        n_sent = sock.send(data[:n_bytes])
        sent_total += n_sent
    return time.time() - t_start


def recv_data(sock, num_bytes, chunk_size):
    received_total = 0
    t_start = time.perf_counter()
    data = bytearray(chunk_size)
    while received_total < num_bytes:
        n_bytes = sock.recv_into(data, min(chunk_size, num_bytes - received_total))
        received_total += n_bytes
    return time.perf_counter() - t_start


def recvall_into(sock, buf):
    """Receive exactly num_bytes of data."""
    num_bytes = len(buf)
    n_read = 0
    buf = memoryview(buf)
    while n_read < num_bytes:
        n_read += sock.recv_into(buf[n_read:])
    return bytes(buf)


def send_pings(sock, num_pings, packet_size):
    data = b'\x00' * packet_size
    ts = []
    buf = bytearray(packet_size)
    for _ in range(num_pings):
        t_start = time.perf_counter()
        sock.sendall(data)
        recvall_into(sock, buf)
        ts.append(time.perf_counter() - t_start)
    return stats(ts)
    
    
def recv_pings(sock, num_pings, packet_size):
    ts = []
    buf = bytearray(packet_size)
    for _ in range(num_pings):
        t_start = time.perf_counter()
        data = recvall_into(sock, buf)
        sock.sendall(data)
        ts.append(time.perf_counter() - t_start)
    return stats(ts)


def run_throughput(host, port, num_bytes, chunk_size):
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
        sock.settimeout(5.0)
        sock.connect((host, port))
        print("Connected to %s:%d" % (host, port))
        sock.send(b'SPDT')
        send_int64(sock, num_bytes)
        send_int64(sock, num_bytes)
        send_int64(sock, chunk_size)
        duration = recv_data(sock, num_bytes, chunk_size)
        print(f"Received {fmt_bytes(num_bytes)} in {duration:.3f} seconds.")
        if duration:
            print(f"Average throughput (download): {fmt_thrpt(num_bytes/duration)}.")
        duration = send_data(sock, num_bytes, chunk_size)
        print(f"Sent {fmt_bytes(num_bytes)} in {duration:.3f} seconds.")
        if duration:
            print(f"Average throughput (upload): {fmt_thrpt(num_bytes/duration)}.")


def run_latency(host, port, num_pings, packet_size):
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
        sock.settimeout(5.0)
        sock.connect((host, port))
        print("Connected to %s:%d" % (host, port))
        sock.send(b'LATN')
        send_int64(sock, num_pings)
        send_int64(sock, num_pings)
        send_int64(sock, packet_size)
        stats = recv_pings(sock, num_pings, packet_size)
        print(f"Latency stats (download):\n{fmt_stats(stats)}.")
        stats = send_pings(sock, num_pings, packet_size)
        print(f"Latency stats (upload):\n{fmt_stats(stats)}.")


def run_shutdown(host, port):
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
        sock.connect((host, port))
        print("Connected to %s:%d" % (host, port))
        sock.send(b'SHUT')

        
def run_server(host, port):
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
        s.bind((host, port))
        s.listen(1)
        print("Listening on %s:%d" % (host, port))
        while True:
            (sock, address) = s.accept()
            print("Incoming connection from {}".format(address))
            try:
                with sock:
                    sock.settimeout(5.0)
                    try:
                        cmd = sock.recv(4)
                    except socket.timeout as e:
                        print("Client did not send a command in time:", e)
                        continue
                    if cmd == b'SPDT':
                        num_bytes_send = recv_int64(sock)
                        num_bytes_recv = recv_int64(sock)
                        chunk_size = recv_int64(sock)
                        print(("Starting speed test with parameters "
                            f"num_bytes_send={num_bytes_send}"
                            f", num_bytes_recv={num_bytes_recv}"
                            f", chunk_size={chunk_size}"))
                        try:
                            send_data(sock, num_bytes_send, chunk_size)
                        except socket.timeout as e:
                            print("Timeout while sending data:", e)
                            continue
                        try:
                            recv_data(sock, num_bytes_recv, chunk_size)
                        except socket.timeout as e:
                            print("Timeout while receiving data:", e)
                            continue
                    elif cmd == b'SHUT':
                        print("Received SHUT command. Shutting down.")
                        return
                    elif cmd == b'LATN':
                        num_pings_send = recv_int64(sock)
                        num_pings_recv = recv_int64(sock)
                        packet_size = recv_int64(sock)
                        print(("Starting latency test with parameters "
                            f"packet_size={packet_size}"
                            f", num_pings_send={num_pings_send}"
                            f", num_pings_recv={num_pings_recv}"))
                        try:
                            send_pings(sock, num_pings_send, packet_size)
                        except socket.timeout as e:
                            print("Timeout while sending pings:", e)
                            continue
                        try:
                            recv_pings(sock, num_pings_recv, packet_size)
                        except socket.timeout as e:
                            print("Timeout while receiving pings:", e)
                            continue
            except OSError as e:
                print("An error occurred:", e)
                
                
if __name__ == "__main__":
    args = parse_args()
    if args.mode == "server":
        run_server(args.host, args.port)
    elif args.command == "throughput":
        run_throughput(args.host, args.port, args.num_bytes, args.chunk_size)
    elif args.command == "shutdown":
        run_shutdown(args.host, args.port)
    elif args.command == "latency":
        run_latency(args.host, args.port, args.num_packets, args.chunk_size)
