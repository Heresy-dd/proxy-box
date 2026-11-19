import json
import sqlite3
import subprocess
import time
import urllib.request
import urllib.parse
import os

DB_PATH = '/data/proxybox/app/data/proxybox.db'
BIN_PATH = '/data/proxybox/app/bin/mihomo'
WORK_DIR = '/tmp/mihomo_udp'
PORT = 29093
SECRET = 'udp123'

os.makedirs(WORK_DIR, exist_ok=True)
conn = sqlite3.connect(DB_PATH)
c = conn.cursor()

def find_node(node_type, name_substr=""):
    rows = c.execute("SELECT id, name, original_name, type, server, port, raw_config FROM nodes WHERE type=? AND original_name LIKE ?", 
                     (node_type, f"%{name_substr}%")).fetchall()
    if rows:
        return {"id": rows[0][0], "name": rows[0][1], "orig": rows[0][2], "type": rows[0][3], "raw": json.loads(rows[0][6])}
    return None

trojan_jp = find_node("trojan", "日本03")
trojan_hk = find_node("trojan", "香港01")
hy2_hk = find_node("hysteria2", "香港01")
hy2_sg = find_node("hysteria2", "新加坡01")

proxies = []
for n, name in [(trojan_jp, "p-trojan-jp"), (trojan_hk, "p-trojan-hk"), (hy2_hk, "p-hy2-hk"), (hy2_sg, "p-hy2-sg")]:
    if n:
        p = dict(n['raw'])
        p['name'] = name
        proxies.append(p)

listeners = []
for idx, p in enumerate(proxies):
    port = 29200 + idx
    listeners.append({
        "name": f"in-{port}",
        "type": "mixed",
        "listen": "0.0.0.0",
        "port": port,
        "proxy": p['name'],
        "udp": True,
        "users": [{"username": "u", "password": "p"}]
    })

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
time.sleep(1)

# Test UDP through socks5 listeners using dig or nslookup via proxy or curl with http3
print("=== TESTING SOCKS5 LISTENERS FOR UDP SUPPORT ===")
for idx, p in enumerate(proxies):
    port = 29200 + idx
    name = p['name']
    # Test HTTP over proxy
    cmd_http = ["curl", "-s", "-m", "5", "--proxy", f"http://u:p@127.0.0.1:{port}", "https://api.ipify.org?format=json"]
    r_http = subprocess.run(cmd_http, capture_output=True, text=True)
    
    # Test DNS resolution over SOCKS5 UDP (curl --socks5-hostname)
    cmd_socks = ["curl", "-s", "-m", "5", "--socks5-hostname", f"u:p@127.0.0.1:{port}", "https://api.ipify.org?format=json"]
    r_socks = subprocess.run(cmd_socks, capture_output=True, text=True)

    print(f"[{name}] Port {port}:")
    print(f"   HTTP proxy: code={r_http.returncode} out={r_http.stdout.strip()}")
    print(f"   SOCKS5 proxy: code={r_socks.returncode} out={r_socks.stdout.strip()}")

proc.terminate()
proc.wait()
log_f.close()
