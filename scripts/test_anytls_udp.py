import socket
import struct
import subprocess
import time
import json
import os

BIN_PATH = '/data/proxybox/app/bin/mihomo'
WORK_DIR = '/tmp/mihomo_anytls_udp'
PORT = 29097
SECRET = 'anyudp123'

os.makedirs(WORK_DIR, exist_ok=True)

p_anytls_macau = {
    "name": "p-anytls-macau",
    "type": "anytls",
    "server": "edge-9850c75ac9a073ed.qianlink.link",
    "port": 24494,
    "password": "78307fb1-caf2-54ac-9d04-7fe101d81808",
    "sni": "vpn.qianlink.link",
    "skip-cert-verify": False,
    "client-fingerprint": "chrome",
    "udp": True
}

cfg = {
    "mode": "rule",
    "log-level": "debug",
    "external-controller": f"127.0.0.1:{PORT}",
    "secret": SECRET,
    "proxies": [p_anytls_macau],
    "listeners": [
        {"name": "in-29601", "type": "mixed", "listen": "0.0.0.0", "port": 29601, "proxy": "p-anytls-macau", "udp": True, "users": [{"username": "u", "password": "p"}]}
    ],
    "rules": ["MATCH,DIRECT"]
}

cfg_path = os.path.join(WORK_DIR, "config.json")
with open(cfg_path, 'w', encoding='utf-8') as f:
    json.dump(cfg, f, indent=2)

log_path = os.path.join(WORK_DIR, "mihomo.log")
log_f = open(log_path, "w", encoding='utf-8')
proc = subprocess.Popen([BIN_PATH, "-d", WORK_DIR, "-f", cfg_path], stdout=log_f, stderr=subprocess.STDOUT)
time.sleep(1)

def test_socks5_udp(proxy_port, user, pwd, target_ip, target_port, payload):
    s_tcp = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s_tcp.settimeout(3)
    s_tcp.connect(('127.0.0.1', proxy_port))

    # Auth handshake
    s_tcp.sendall(b'\x05\x02\x00\x02')
    res = s_tcp.recv(2)
    if res[1] == 2:
        req = b'\x01' + bytes([len(user)]) + user.encode() + bytes([len(pwd)]) + pwd.encode()
        s_tcp.sendall(req)
        auth_res = s_tcp.recv(2)
        if auth_res[1] != 0:
            raise Exception("Auth failed")

    # UDP ASSOCIATE request: CMD=3
    s_tcp.sendall(b'\x05\x03\x00\x01\x00\x00\x00\x00\x00\x00')
    resp = s_tcp.recv(10)
    if resp[1] != 0:
        raise Exception(f"UDP Associate failed with code {resp[1]}")

    if resp[3] == 1:
        relay_ip = socket.inet_ntoa(resp[4:8])
        relay_port = struct.unpack('!H', resp[8:10])[0]
    else:
        relay_ip = '127.0.0.1'
        relay_port = struct.unpack('!H', resp[-2:])[0]

    s_udp = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    s_udp.settimeout(3)
    header = b'\x00\x00\x00\x01' + socket.inet_aton(target_ip) + struct.pack('!H', target_port)
    packet = header + payload
    s_udp.sendto(packet, (relay_ip, relay_port))
    data, addr = s_udp.recvfrom(2048)
    s_tcp.close()
    s_udp.close()
    return True, len(data)

dns_query = b'\xaa\xbb\x01\x00\x00\x01\x00\x00\x00\x00\x00\x00\x07example\x03com\x00\x00\x01\x00\x01'
try:
    ok, sz = test_socks5_udp(29601, "u", "p", "1.1.1.1", 53, dns_query)
    print(f"Direct AnyTLS UDP test: SUCCESS ({sz} bytes)")
except Exception as e:
    print(f"Direct AnyTLS UDP test: FAILED ({e})")

proc.terminate()
proc.wait()
log_f.close()
