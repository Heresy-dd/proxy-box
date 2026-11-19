import json
import subprocess
import time
import urllib.request
import urllib.parse
import os

BIN_PATH = '/data/proxybox/app/bin/mihomo'
WORK_DIR = '/tmp/mihomo_anytls'
PORT = 29094
SECRET = 'any123'

os.makedirs(WORK_DIR, exist_ok=True)

proxy_anytls = {
    "name": "qianlink-anytls-macau",
    "type": "anytls",
    "server": "edge-9850c75ac9a073ed.qianlink.link",
    "port": 24494,
    "password": "78307fb1-caf2-54ac-9d04-7fe101d81808",
    "sni": "vpn.qianlink.link",
    "skip-cert-verify": False,
    "client-fingerprint": "chrome",
    "udp": True
}

# Also chain it with Trojan
proxy_trojan = {
    "name": "p-trojan",
    "type": "trojan",
    "server": "35.77.107.225",
    "port": 443,
    "password": "5a36317c-093e-ea5c-ca0b-6cfb05f4a6db", # from db
    "sni": "www.bing.com",
    "skip-cert-verify": True,
    "udp": True
}

# Read real password for trojan from db
import sqlite3
c = sqlite3.connect('/data/proxybox/app/data/proxybox.db').cursor()
trojan_raw = json.loads(c.execute("SELECT raw_config FROM nodes WHERE type='trojan' AND original_name LIKE '%日本03%'").fetchone()[0])
proxy_trojan = trojan_raw
proxy_trojan['name'] = 'p-trojan'

chain_anytls = dict(proxy_anytls)
chain_anytls['name'] = 'chain-trojan-anytls'
chain_anytls['dialer-proxy'] = 'p-trojan'

cfg = {
    "mode": "rule",
    "log-level": "debug",
    "external-controller": f"127.0.0.1:{PORT}",
    "secret": SECRET,
    "proxies": [proxy_anytls, proxy_trojan, chain_anytls],
    "listeners": [
        {
            "name": "in-29301",
            "type": "mixed",
            "listen": "0.0.0.0",
            "port": 29301,
            "proxy": "qianlink-anytls-macau",
            "udp": True,
            "users": [{"username": "u", "password": "p"}]
        },
        {
            "name": "in-29302",
            "type": "mixed",
            "listen": "0.0.0.0",
            "port": 29302,
            "proxy": "chain-trojan-anytls",
            "udp": True,
            "users": [{"username": "u", "password": "p"}]
        }
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

def get_delay(name, timeout=8000):
    enc_name = urllib.parse.quote(name)
    url = urllib.parse.quote("https://cp.cloudflare.com/generate_204")
    api_url = f"http://127.0.0.1:{PORT}/proxies/{enc_name}/delay?url={url}&timeout={timeout}"
    req = urllib.request.Request(api_url, headers={"Authorization": f"Bearer {SECRET}"})
    try:
        with urllib.request.urlopen(req, timeout=timeout/1000 + 2) as resp:
            return json.loads(resp.read().decode('utf-8'))
    except Exception as e:
        return {"error": str(e)}

def curl_listener(port):
    cmd = ["curl", "-s", "-m", "10", "--proxy", f"http://u:p@127.0.0.1:{port}", "https://api.ipify.org?format=json"]
    res = subprocess.run(cmd, capture_output=True, text=True)
    return {"code": res.returncode, "out": res.stdout.strip()}

print("=== DELAY CHECKS ===")
print("Direct AnyTLS:", get_delay("qianlink-anytls-macau"))
print("Chain Trojan -> AnyTLS:", get_delay("chain-trojan-anytls"))

print("=== CURL LISTENERS ===")
print("Curl Direct AnyTLS (29301):", curl_listener(29301))
print("Curl Chain Trojan -> AnyTLS (29302):", curl_listener(29302))

proc.terminate()
proc.wait()
log_f.close()

print("\n=== DEBUG LOGS ===")
with open(log_path, 'r', encoding='utf-8', errors='ignore') as f:
    for line in f:
        line = line.strip()
        if any(w in line.lower() for w in ["anytls", "error", "warn", "fail"]):
            print(line)
