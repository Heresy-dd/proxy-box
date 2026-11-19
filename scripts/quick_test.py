import json
import sqlite3
import subprocess
import time
import urllib.request
import urllib.parse
import os

DB_PATH = '/data/proxybox/app/data/proxybox.db'
BIN_PATH = '/data/proxybox/app/bin/mihomo'
WORK_DIR = '/tmp/mihomo_quick'
PORT = 29091
SECRET = 'quick123'

os.makedirs(WORK_DIR, exist_ok=True)
conn = sqlite3.connect(DB_PATH)
c = conn.cursor()

def find_node(node_type, name_substr=""):
    rows = c.execute("SELECT id, name, original_name, type, server, port, raw_config FROM nodes WHERE type=? AND original_name LIKE ?", 
                     (node_type, f"%{name_substr}%")).fetchall()
    if rows:
        return {
            "id": rows[0][0],
            "name": rows[0][1],
            "original_name": rows[0][2],
            "type": rows[0][3],
            "server": rows[0][4],
            "port": rows[0][5],
            "raw_config": json.loads(rows[0][6])
        }
    return None

tuic = find_node("tuic", "澳门")
trojan = find_node("trojan", "日本03")
if not trojan:
    trojan = find_node("trojan", "")
hy2 = find_node("hysteria2", "香港01")
vless = find_node("vless", "新加坡01")

proxies = []

# Standalone nodes
p_tuic = dict(tuic['raw_config'])
p_tuic['name'] = 'test-tuic-direct'
proxies.append(p_tuic)

p_trojan = dict(trojan['raw_config'])
p_trojan['name'] = 'test-trojan-direct'
proxies.append(p_trojan)

p_hy2 = dict(hy2['raw_config'])
p_hy2['name'] = 'test-hy2-direct'
proxies.append(p_hy2)

p_vless = dict(vless['raw_config'])
p_vless['name'] = 'test-vless-direct'
proxies.append(p_vless)

# Chained TUIC
c_trojan_tuic = dict(tuic['raw_config'])
c_trojan_tuic['name'] = 'test-chain-trojan-tuic'
c_trojan_tuic['dialer-proxy'] = 'test-trojan-direct'
proxies.append(c_trojan_tuic)

c_hy2_tuic = dict(tuic['raw_config'])
c_hy2_tuic['name'] = 'test-chain-hy2-tuic'
c_hy2_tuic['dialer-proxy'] = 'test-hy2-direct'
proxies.append(c_hy2_tuic)

listeners = [
    {
        "name": "in-29101",
        "type": "mixed",
        "listen": "0.0.0.0",
        "port": 29101,
        "proxy": "test-tuic-direct",
        "udp": True,
        "users": [{"username": "u", "password": "p"}]
    },
    {
        "name": "in-29102",
        "type": "mixed",
        "listen": "0.0.0.0",
        "port": 29102,
        "proxy": "test-trojan-direct",
        "udp": True,
        "users": [{"username": "u", "password": "p"}]
    },
    {
        "name": "in-29103",
        "type": "mixed",
        "listen": "0.0.0.0",
        "port": 29103,
        "proxy": "test-hy2-direct",
        "udp": True,
        "users": [{"username": "u", "password": "p"}]
    },
    {
        "name": "in-29104",
        "type": "mixed",
        "listen": "0.0.0.0",
        "port": 29104,
        "proxy": "test-chain-trojan-tuic",
        "udp": True,
        "users": [{"username": "u", "password": "p"}]
    },
    {
        "name": "in-29105",
        "type": "mixed",
        "listen": "0.0.0.0",
        "port": 29105,
        "proxy": "test-chain-hy2-tuic",
        "udp": True,
        "users": [{"username": "u", "password": "p"}]
    }
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

# Wait ready
time.sleep(1)

def get_delay(name, url="https://cp.cloudflare.com/generate_204", timeout=5000):
    enc_name = urllib.parse.quote(name)
    enc_url = urllib.parse.quote(url)
    api_url = f"http://127.0.0.1:{PORT}/proxies/{enc_name}/delay?url={enc_url}&timeout={timeout}"
    req = urllib.request.Request(api_url, headers={"Authorization": f"Bearer {SECRET}"})
    t0 = time.time()
    try:
        with urllib.request.urlopen(req, timeout=timeout/1000 + 2) as resp:
            data = json.loads(resp.read().decode('utf-8'))
            return {"ok": True, "delay": data.get("delay"), "elapsed": int((time.time() - t0)*1000)}
    except urllib.error.HTTPError as e:
        return {"ok": False, "status": e.code, "body": e.read().decode('utf-8', errors='ignore'), "elapsed": int((time.time() - t0)*1000)}
    except Exception as e:
        return {"ok": False, "error": str(e), "elapsed": int((time.time() - t0)*1000)}

def curl_listener(port, url="https://api.ipify.org?format=json"):
    cmd = ["curl", "-v", "-m", "5", "--proxy", f"http://u:p@127.0.0.1:{port}", url]
    res = subprocess.run(cmd, capture_output=True, text=True)
    return {
        "code": res.returncode,
        "stdout": res.stdout.strip(),
        "stderr_tail": "\n".join(res.stderr.strip().splitlines()[-4:])
    }

print("=== 1. MIHOMO API DELAY CHECKS ===")
for p in ["test-tuic-direct", "test-trojan-direct", "test-hy2-direct", "test-vless-direct", "test-chain-trojan-tuic", "test-chain-hy2-tuic"]:
    r = get_delay(p, timeout=5000)
    print(f"Delay [{p}]: {r}")

print("\n=== 2. CURL LISTENER CHECKS ===")
for port, name in [
    (29101, "TUIC Direct"),
    (29102, "Trojan Direct"),
    (29103, "Hy2 Direct"),
    (29104, "Trojan -> TUIC Chain"),
    (29105, "Hy2 -> TUIC Chain"),
]:
    r = curl_listener(port)
    print(f"Curl [{port} - {name}]: code={r['code']} body={r['stdout']}")
    if r['code'] != 0:
        print(f"   stderr: {r['stderr_tail']}")

proc.terminate()
proc.wait()
log_f.close()

print("\n=== 3. MIHOMO DEBUG LOGS ===")
with open(log_path, 'r', encoding='utf-8', errors='ignore') as f:
    for line in f:
        line = line.strip()
        if any(w in line.lower() for w in ["error", "warn", "fail", "context deadline", "quic", "dial", "read addr"]):
            print(line)
