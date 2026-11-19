import json
import subprocess
import time
import urllib.request
import urllib.parse
import os
import sqlite3

BIN_PATH = '/data/proxybox/app/bin/mihomo'
WORK_DIR = '/tmp/mihomo_anytls_hy2'
PORT = 29095
SECRET = 'anyhy2'

os.makedirs(WORK_DIR, exist_ok=True)

c = sqlite3.connect('/data/proxybox/app/data/proxybox.db').cursor()
hy2_raw = json.loads(c.execute("SELECT raw_config FROM nodes WHERE type='hysteria2' AND original_name LIKE '%香港01%'").fetchone()[0])
hy2_raw['name'] = 'p-hy2'

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

chain_hy2_anytls = dict(proxy_anytls)
chain_hy2_anytls['name'] = 'chain-hy2-anytls'
chain_hy2_anytls['dialer-proxy'] = 'p-hy2'

cfg = {
    "mode": "rule",
    "log-level": "info",
    "external-controller": f"127.0.0.1:{PORT}",
    "secret": SECRET,
    "proxies": [proxy_anytls, hy2_raw, chain_hy2_anytls],
    "listeners": [
        {
            "name": "in-29402",
            "type": "mixed",
            "listen": "0.0.0.0",
            "port": 29402,
            "proxy": "chain-hy2-anytls",
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

print("Delay Chain Hy2 -> AnyTLS:", get_delay("chain-hy2-anytls"))
print("Curl Chain Hy2 -> AnyTLS (29402):", curl_listener(29402))

proc.terminate()
proc.wait()
log_f.close()
