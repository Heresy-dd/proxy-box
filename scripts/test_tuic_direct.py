import json
import sqlite3
import subprocess
import time
import urllib.request
import urllib.parse
import os

DB_PATH = '/data/proxybox/app/data/proxybox.db'
BIN_PATH = '/data/proxybox/app/bin/mihomo'
WORK_DIR = '/tmp/mihomo_tuic'
PORT = 29092
SECRET = 'tuic123'

os.makedirs(WORK_DIR, exist_ok=True)
conn = sqlite3.connect(DB_PATH)
c = conn.cursor()

def get_node_by_name(sub_str):
    r = c.execute("SELECT id, name, original_name, type, raw_config FROM nodes WHERE original_name LIKE ?", (f"%{sub_str}%",)).fetchone()
    if r:
        return {"id": r[0], "name": r[1], "orig": r[2], "type": r[3], "raw": json.loads(r[4])}
    return None

macau = get_node_by_name("澳门")
hk = get_node_by_name("香港WiFicall")
ix_hy2 = get_node_by_name("IX专线直通")
japan = get_node_by_name("日本（动态家庭）")

proxies = []
for node, alias in [(macau, "tuic-macau"), (hk, "tuic-hk"), (japan, "tuic-japan"), (ix_hy2, "hy2-ix")]:
    if node:
        p = dict(node['raw'])
        p['name'] = alias
        proxies.append(p)

cfg = {
    "mode": "rule",
    "log-level": "debug",
    "external-controller": f"127.0.0.1:{PORT}",
    "secret": SECRET,
    "proxies": proxies,
    "rules": ["MATCH,DIRECT"]
}

cfg_path = os.path.join(WORK_DIR, "config.json")
with open(cfg_path, 'w', encoding='utf-8') as f:
    json.dump(cfg, f, indent=2)

log_path = os.path.join(WORK_DIR, "mihomo.log")
log_f = open(log_path, "w", encoding='utf-8')
proc = subprocess.Popen([BIN_PATH, "-d", WORK_DIR, "-f", cfg_path], stdout=log_f, stderr=subprocess.STDOUT)
time.sleep(1)

def get_delay(name, timeout_ms=15000):
    enc_name = urllib.parse.quote(name)
    url = urllib.parse.quote("https://cp.cloudflare.com/generate_204")
    api_url = f"http://127.0.0.1:{PORT}/proxies/{enc_name}/delay?url={url}&timeout={timeout_ms}"
    req = urllib.request.Request(api_url, headers={"Authorization": f"Bearer {SECRET}"})
    t0 = time.time()
    try:
        with urllib.request.urlopen(req, timeout=timeout_ms/1000 + 2) as resp:
            data = json.loads(resp.read().decode('utf-8'))
            return {"ok": True, "delay": data.get("delay"), "elapsed": int((time.time() - t0)*1000)}
    except urllib.error.HTTPError as e:
        return {"ok": False, "status": e.code, "body": e.read().decode('utf-8', errors='ignore'), "elapsed": int((time.time() - t0)*1000)}
    except Exception as e:
        return {"ok": False, "error": str(e), "elapsed": int((time.time() - t0)*1000)}

print("=== TESTING DIRECT QIANLINK NODES WITH 15s TIMEOUT ===")
for p in ["tuic-macau", "tuic-hk", "tuic-japan", "hy2-ix"]:
    r = get_delay(p, timeout_ms=15000)
    print(f"Direct [{p}]: {r}")

proc.terminate()
proc.wait()
log_f.close()

print("\n=== DEBUG LOGS ===")
with open(log_path, 'r', encoding='utf-8', errors='ignore') as f:
    for line in f:
        line = line.strip()
        if any(w in line.lower() for w in ["error", "warn", "fail", "context deadline", "tuic", "handshake"]):
            print(line)
