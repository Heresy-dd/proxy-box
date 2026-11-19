import json
import sqlite3
import subprocess
import time
import urllib.request
import urllib.parse
import socket
import struct
import os

BIN_PATH = '/data/proxybox/app/bin/mihomo'
WORK_DIR = '/tmp/mihomo_quic_verify'
PORT = 29096
SECRET = 'quicverify123'

os.makedirs(WORK_DIR, exist_ok=True)
c = sqlite3.connect('/data/proxybox/app/data/proxybox.db').cursor()

def get_node(sub_prefix, node_type, name_substr):
    r = c.execute("SELECT id, name, original_name, type, server, port, raw_config FROM nodes WHERE type=? AND original_name LIKE ?", 
                  (node_type, f"%{name_substr}%")).fetchall()
    for row in r:
        if sub_prefix in row[1]:
            return {"id": row[0], "name": row[1], "orig": row[2], "type": row[3], "server": row[4], "port": row[5], "raw": json.loads(row[6])}
    return None

trojan_jp = get_node("宝可梦", "trojan", "日本03")
hy2_hk = get_node("宝可梦", "hysteria2", "香港01")
hy2_sg = get_node("宝可梦", "hysteria2", "新加坡01")

print(f"Trojan JP: {trojan_jp['orig'] if trojan_jp else 'None'}")
print(f"Hy2 HK: {hy2_hk['orig'] if hy2_hk else 'None'}")
print(f"Hy2 SG: {hy2_sg['orig'] if hy2_sg else 'None'}")

# Setup Proxies
p_trojan = dict(trojan_jp['raw'])
p_trojan['name'] = 'p-trojan-jp'
p_trojan.pop('dialer-proxy', None)

p_hy2_hk = dict(hy2_hk['raw'])
p_hy2_hk['name'] = 'p-hy2-hk'
p_hy2_hk.pop('dialer-proxy', None)

p_hy2_sg = dict(hy2_sg['raw'])
p_hy2_sg['name'] = 'p-hy2-sg'
p_hy2_sg.pop('dialer-proxy', None)

# Chain 1: Trojan JP -> Hy2 HK
c_trojan_hy2 = dict(hy2_hk['raw'])
c_trojan_hy2['name'] = 'chain-trojan-hy2'
c_trojan_hy2['dialer-proxy'] = 'p-trojan-jp'

# Chain 2: Hy2 HK -> Hy2 SG
c_hy2_hy2 = dict(hy2_sg['raw'])
c_hy2_hy2['name'] = 'chain-hy2-hy2'
c_hy2_hy2['dialer-proxy'] = 'p-hy2-hk'

# Also AnyTLS Macau node for UDP test
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

# Chain 3: Trojan JP -> AnyTLS Macau
c_trojan_anytls = dict(p_anytls_macau)
c_trojan_anytls['name'] = 'chain-trojan-anytls'
c_trojan_anytls['dialer-proxy'] = 'p-trojan-jp'

proxies = [p_trojan, p_hy2_hk, p_hy2_sg, c_trojan_hy2, c_hy2_hy2, p_anytls_macau, c_trojan_anytls]

listeners = [
    {"name": "in-29501", "type": "mixed", "listen": "0.0.0.0", "port": 29501, "proxy": "p-trojan-jp", "udp": True, "users": [{"username": "u", "password": "p"}]},
    {"name": "in-29502", "type": "mixed", "listen": "0.0.0.0", "port": 29502, "proxy": "p-hy2-hk", "udp": True, "users": [{"username": "u", "password": "p"}]},
    {"name": "in-29503", "type": "mixed", "listen": "0.0.0.0", "port": 29503, "proxy": "p-hy2-sg", "udp": True, "users": [{"username": "u", "password": "p"}]},
    {"name": "in-29504", "type": "mixed", "listen": "0.0.0.0", "port": 29504, "proxy": "chain-trojan-hy2", "udp": True, "users": [{"username": "u", "password": "p"}]},
    {"name": "in-29505", "type": "mixed", "listen": "0.0.0.0", "port": 29505, "proxy": "chain-hy2-hy2", "udp": True, "users": [{"username": "u", "password": "p"}]},
    {"name": "in-29506", "type": "mixed", "listen": "0.0.0.0", "port": 29506, "proxy": "chain-trojan-anytls", "udp": True, "users": [{"username": "u", "password": "p"}]},
]

cfg = {
    "mode": "rule",
    "log-level": "debug",
    "external-controller": f"127.0.0.1:{PORT}",
    "secret": SECRET,
    "proxies": proxies,
    "listeners": listeners,
    "rules": ["MATCH,DIRECT"]
}

cfg_path = os.path.join(WORK_DIR, "config.json")
with open(cfg_path, 'w', encoding='utf-8') as f:
    json.dump(cfg, f, indent=2)

log_path = os.path.join(WORK_DIR, "mihomo.log")
log_f = open(log_path, "w", encoding='utf-8')
proc = subprocess.Popen([BIN_PATH, "-d", WORK_DIR, "-f", cfg_path], stdout=log_f, stderr=subprocess.STDOUT)
time.sleep(1.5)

def get_delay(name, timeout=10000):
    enc_name = urllib.parse.quote(name)
    url = urllib.parse.quote("https://cp.cloudflare.com/generate_204")
    api_url = f"http://127.0.0.1:{PORT}/proxies/{enc_name}/delay?url={url}&timeout={timeout}"
    req = urllib.request.Request(api_url, headers={"Authorization": f"Bearer {SECRET}"})
    t0 = time.time()
    try:
        with urllib.request.urlopen(req, timeout=timeout/1000 + 2) as resp:
            data = json.loads(resp.read().decode('utf-8'))
            return {"ok": True, "delay": data.get("delay"), "elapsed": int((time.time()-t0)*1000)}
    except urllib.error.HTTPError as e:
        return {"ok": False, "status": e.code, "body": e.read().decode('utf-8', errors='ignore'), "elapsed": int((time.time()-t0)*1000)}
    except Exception as e:
        return {"ok": False, "error": str(e), "elapsed": int((time.time()-t0)*1000)}

def curl_listener(port):
    cmd = ["curl", "-s", "-m", "10", "--proxy", f"http://u:p@127.0.0.1:{port}", "https://api.ipify.org?format=json"]
    t0 = time.time()
    res = subprocess.run(cmd, capture_output=True, text=True)
    return {"code": res.returncode, "out": res.stdout.strip(), "elapsed": int((time.time()-t0)*1000)}

print("\n=== A1. BASELINE DIRECT LATENCY & EGRESS ===")
for name, port in [("p-trojan-jp", 29501), ("p-hy2-hk", 29502), ("p-hy2-sg", 29503)]:
    d = get_delay(name)
    c = curl_listener(port)
    print(f"[{name}] delay={d} | curl={c}")

print("\n=== A1. QUIC CHAINING EXPERIMENT ===")
for name, port in [("chain-trojan-hy2", 29504), ("chain-hy2-hy2", 29505)]:
    d = get_delay(name)
    c = curl_listener(port)
    print(f"[{name}] delay={d} | curl={c}")

print("\n=== A4. SOCKS5 UDP TEST ON TROJAN -> ANYTLS CHAIN (PORT 29506) ===")
# Test HTTP egress
c_any = curl_listener(29506)
print(f"HTTP egress [chain-trojan-anytls]: {c_any}")

# Test SOCKS5 UDP DNS (UDP port 53 query to 1.1.1.1)
def test_socks5_udp(proxy_port, user, pwd, target_ip, target_port, payload):
    s_tcp = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s_tcp.settimeout(5)
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
    s_udp.settimeout(5)
    header = b'\x00\x00\x00\x01' + socket.inet_aton(target_ip) + struct.pack('!H', target_port)
    packet = header + payload
    s_udp.sendto(packet, (relay_ip, relay_port))
    data, addr = s_udp.recvfrom(2048)
    s_tcp.close()
    s_udp.close()
    return True, len(data)

dns_query = b'\xaa\xbb\x01\x00\x00\x01\x00\x00\x00\x00\x00\x00\x07example\x03com\x00\x00\x01\x00\x01'
try:
    ok, sz = test_socks5_udp(29506, "u", "p", "1.1.1.1", 53, dns_query)
    print(f"SOCKS5 UDP DNS through [Trojan -> AnyTLS -> Port 29506]: SUCCESS! Received {sz} bytes")
except Exception as e:
    print(f"SOCKS5 UDP DNS through [Trojan -> AnyTLS -> Port 29506]: FAILED! {e}")

proc.terminate()
proc.wait()
log_f.close()

print("\n=== DEBUG LOG SNIPPETS FOR CHAINING ===")
with open(log_path, 'r', encoding='utf-8', errors='ignore') as f:
    for line in f:
        line = line.strip()
        if any(w in line.lower() for w in ["chain-trojan-hy2", "chain-hy2-hy2", "error", "warn", "using"]):
            print(line)
