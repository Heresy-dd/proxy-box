import os
import sys
import json
import sqlite3
import subprocess
import time
import urllib.request
import urllib.parse
import shutil

DB_PATH = '/data/proxybox/app/data/proxybox.db'
BIN_PATH = '/data/proxybox/app/bin/mihomo'
WORK_DIR = '/tmp/mihomo_diag'
PORT = 29090
SECRET = 'diagsecret123'

os.makedirs(WORK_DIR, exist_ok=True)

conn = sqlite3.connect(DB_PATH)
c = conn.cursor()

def find_node(sub_prefix, node_type, name_substr=""):
    rows = c.execute("SELECT id, name, original_name, type, server, port, raw_config FROM nodes WHERE type=? AND original_name LIKE ?", 
                     (node_type, f"%{name_substr}%")).fetchall()
    for r in rows:
        if sub_prefix in r[1]:
            return {
                "id": r[0],
                "name": r[1],
                "original_name": r[2],
                "type": r[3],
                "server": r[4],
                "port": r[5],
                "raw_config": json.loads(r[6])
            }
    if rows:
        r = rows[0]
        return {
            "id": r[0],
            "name": r[1],
            "original_name": r[2],
            "type": r[3],
            "server": r[4],
            "port": r[5],
            "raw_config": json.loads(r[6])
        }
    return None

print("Finding test nodes...")
tuic_macau = find_node("千链", "tuic", "澳门")
tuic_hk = find_node("千链", "tuic", "香港")
trojan_node = find_node("宝可梦", "trojan", "日本")
if not trojan_node:
    trojan_node = find_node("宝可梦", "trojan", "")
hy2_node = find_node("宝可梦", "hysteria2", "香港01")
if not hy2_node:
    hy2_node = find_node("宝可梦", "hysteria2", "")
vless_node = find_node("宝可梦", "vless", "新加坡")

print(f"TUIC Macau: {tuic_macau['name'] if tuic_macau else 'None'}")
print(f"TUIC HK: {tuic_hk['name'] if tuic_hk else 'None'}")
print(f"Trojan: {trojan_node['name'] if trojan_node else 'None'}")
print(f"Hysteria2: {hy2_node['name'] if hy2_node else 'None'}")
print(f"VLESS: {vless_node['name'] if vless_node else 'None'}")

# Generate test config
proxies_list = []

# 1. Direct TUIC Macau
p1 = dict(tuic_macau['raw_config'])
p1['name'] = 'direct-tuic-macau'
p1.pop('dialer-proxy', None)
proxies_list.append(p1)

# 2. Direct TUIC HK
p2 = dict(tuic_hk['raw_config'])
p2['name'] = 'direct-tuic-hk'
p2.pop('dialer-proxy', None)
proxies_list.append(p2)

# 3. Direct Trojan
p3 = dict(trojan_node['raw_config'])
p3['name'] = 'direct-trojan'
p3.pop('dialer-proxy', None)
proxies_list.append(p3)

# 4. Direct Hysteria2
p4 = dict(hy2_node['raw_config'])
p4['name'] = 'direct-hy2'
p4.pop('dialer-proxy', None)
proxies_list.append(p4)

# 5. Direct VLESS
if vless_node:
    p5 = dict(vless_node['raw_config'])
    p5['name'] = 'direct-vless'
    p5.pop('dialer-proxy', None)
    proxies_list.append(p5)

# 6. Chain: Trojan -> TUIC Macau
c1 = dict(tuic_macau['raw_config'])
c1['name'] = 'chain-trojan-tuic-macau'
c1['dialer-proxy'] = 'direct-trojan'
proxies_list.append(c1)

# 7. Chain: Hysteria2 -> TUIC Macau
c2 = dict(tuic_macau['raw_config'])
c2['name'] = 'chain-hy2-tuic-macau'
c2['dialer-proxy'] = 'direct-hy2'
proxies_list.append(c2)

# 8. Chain: Trojan -> TUIC HK
c3 = dict(tuic_hk['raw_config'])
c3['name'] = 'chain-trojan-tuic-hk'
c3['dialer-proxy'] = 'direct-trojan'
proxies_list.append(c3)

# 9. Chain: Hysteria2 -> TUIC HK
c4 = dict(tuic_hk['raw_config'])
c4['name'] = 'chain-hy2-tuic-hk'
c4['dialer-proxy'] = 'direct-hy2'
proxies_list.append(c4)

# Listeners
listeners_list = [
    {
        "name": "in-29001",
        "type": "mixed",
        "listen": "0.0.0.0",
        "port": 29001,
        "proxy": "direct-tuic-macau",
        "udp": True,
        "users": [{"username": "testuser", "password": "testpassword"}]
    },
    {
        "name": "in-29002",
        "type": "mixed",
        "listen": "0.0.0.0",
        "port": 29002,
        "proxy": "direct-trojan",
        "udp": True,
        "users": [{"username": "testuser", "password": "testpassword"}]
    },
    {
        "name": "in-29003",
        "type": "mixed",
        "listen": "0.0.0.0",
        "port": 29003,
        "proxy": "chain-trojan-tuic-macau",
        "udp": True,
        "users": [{"username": "testuser", "password": "testpassword"}]
    },
    {
        "name": "in-29004",
        "type": "mixed",
        "listen": "0.0.0.0",
        "port": 29004,
        "proxy": "chain-hy2-tuic-macau",
        "udp": True,
        "users": [{"username": "testuser", "password": "testpassword"}]
    }
]

diag_cfg = {
    "mode": "rule",
    "log-level": "debug",
    "external-controller": f"127.0.0.1:{PORT}",
    "secret": SECRET,
    "proxies": proxies_list,
    "listeners": listeners_list,
    "rules": ["MATCH,DIRECT"]
}

cfg_path = os.path.join(WORK_DIR, "config.json")
with open(cfg_path, 'w', encoding='utf-8') as f:
    json.dump(diag_cfg, f, indent=2, ensure_ascii=False)

print(f"Wrote config to {cfg_path}")

# Run config test
t_res = subprocess.run([BIN_PATH, "-t", "-d", WORK_DIR, "-f", cfg_path], capture_output=True, text=True)
print(f"Config check: returncode={t_res.returncode}")
if t_res.returncode != 0:
    print("STDOUT:", t_res.stdout)
    print("STDERR:", t_res.stderr)
    sys.exit(1)

# Start mihomo in background with stdout/stderr piped to a log file
log_path = os.path.join(WORK_DIR, "mihomo.log")
log_file = open(log_path, "w", encoding='utf-8')
proc = subprocess.Popen([BIN_PATH, "-d", WORK_DIR, "-f", cfg_path], stdout=log_file, stderr=subprocess.STDOUT)
print(f"Started mihomo PID={proc.pid}")

# Wait for controller ready
ready = False
for _ in range(30):
    time.sleep(0.2)
    try:
        req = urllib.request.Request(f"http://127.0.0.1:{PORT}/version", headers={"Authorization": f"Bearer {SECRET}"})
        with urllib.request.urlopen(req, timeout=1) as resp:
            if resp.status == 200:
                print(f"Controller ready! Version info: {resp.read().decode('utf-8')}")
                ready = True
                break
    except Exception:
        pass

if not ready:
    print("Failed to start or connect to controller!")
    proc.terminate()
    sys.exit(1)

def test_proxy_delay(name, test_url, timeout_ms=10000):
    enc_name = urllib.parse.quote(name)
    enc_url = urllib.parse.quote(test_url)
    api_url = f"http://127.0.0.1:{PORT}/proxies/{enc_name}/delay?url={enc_url}&timeout={timeout_ms}"
    req = urllib.request.Request(api_url, headers={"Authorization": f"Bearer {SECRET}"})
    t0 = time.time()
    try:
        with urllib.request.urlopen(req, timeout=(timeout_ms/1000.0 + 3)) as resp:
            data = json.loads(resp.read().decode('utf-8'))
            elapsed = int((time.time() - t0) * 1000)
            return {"ok": True, "delay": data.get("delay"), "elapsed_ms": elapsed, "raw": data}
    except urllib.error.HTTPError as e:
        body = e.read().decode('utf-8', errors='ignore')
        elapsed = int((time.time() - t0) * 1000)
        return {"ok": False, "status": e.code, "body": body, "elapsed_ms": elapsed}
    except Exception as e:
        elapsed = int((time.time() - t0) * 1000)
        return {"ok": False, "error": str(e), "elapsed_ms": elapsed}

urls = [
    "https://cp.cloudflare.com/generate_204",
    "http://www.gstatic.com/generate_204",
]

test_targets = [
    "direct-tuic-macau",
    "direct-tuic-hk",
    "direct-trojan",
    "direct-hy2",
    "direct-vless",
    "chain-trojan-tuic-macau",
    "chain-hy2-tuic-macau",
    "chain-trojan-tuic-hk",
    "chain-hy2-tuic-hk"
]

print("\n" + "="*70)
print("LAYERED DELAY TESTS (timeout=12000ms)")
print("="*70)

results = {}
for target in test_targets:
    print(f"\n--- Testing proxy: {target} ---")
    for url in urls:
        res = test_proxy_delay(target, url, timeout_ms=12000)
        print(f"  URL: {url} -> {res}")
        results[f"{target} | {url}"] = res

print("\n" + "="*70)
print("TESTING CURL ON LISTENERS")
print("="*70)

def test_curl_listener(port, url="https://api.ipify.org?format=json"):
    cmd = [
        "curl", "-v", "-m", "15",
        "--proxy", f"http://testuser:testpassword@127.0.0.1:{port}",
        url
    ]
    t0 = time.time()
    res = subprocess.run(cmd, capture_output=True, text=True)
    elapsed = int((time.time() - t0) * 1000)
    return {
        "port": port,
        "returncode": res.returncode,
        "stdout": res.stdout,
        "stderr": res.stderr[-500:], # last 500 chars
        "elapsed_ms": elapsed
    }

for p in [29001, 29002, 29003, 29004]:
    print(f"\n--- Testing Listener Port {p} with curl ---")
    c_res = test_curl_listener(p)
    print(f"Port {p} code={c_res['returncode']} elapsed={c_res['elapsed_ms']}ms")
    print(f"Stdout: {c_res['stdout'][:200]}")
    print(f"Stderr snippet:\n{c_res['stderr']}")

# Also test production port 9001
print("\n--- Testing Production Port 9001 with curl ---")
prod_res = subprocess.run([
    "curl", "-v", "-m", "15",
    "--proxy", "http://pb_5a0f0641:0c3368cbc56259ab@127.0.0.1:9001",
    "https://api.ipify.org?format=json"
], capture_output=True, text=True)
print(f"Prod 9001 code={prod_res.returncode}")
print(f"Prod stdout: {prod_res.stdout}")
print(f"Prod stderr snippet:\n{prod_res.stderr[-500:]}")

# Flush and stop
proc.terminate()
proc.wait()
log_file.close()

print("\n" + "="*70)
print("MIHOMO DEBUG LOG EXTRACTS")
print("="*70)
with open(log_path, "r", encoding='utf-8', errors='ignore') as f:
    lines = f.readlines()
    for line in lines:
        line_clean = line.strip()
        if any(k in line_clean.lower() for k in ["error", "warn", "fail", "timeout", "dial", "handshake", "chain", "tuic", "generate_204", "ipify"]):
            print(line_clean)
