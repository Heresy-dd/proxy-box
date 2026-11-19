import urllib.request
import ssl
import json
import re
import subprocess
import time
import os
import sqlite3
import yaml
from concurrent.futures import ThreadPoolExecutor

BIN_PATH = '/data/proxybox/app/bin/mihomo'
WORK_DIR = '/tmp/mihomo_matrix'
PORT = 29098
SECRET = 'matrix123'

os.makedirs(WORK_DIR, exist_ok=True)

# 1. Fetch native subscription
url = "https://sub.qianlink.link/customer-sub/LGPbtwU1i9SZGVtbGIBa-5UyfzZuXcj7/qianlink.heresy"
req = urllib.request.Request(url, headers={"User-Agent": "clash.meta"})
ctx = ssl.create_default_context()
ctx.check_hostname = False
ctx.verify_mode = ssl.CERT_NONE

with urllib.request.urlopen(req, context=ctx, timeout=15) as resp:
    parsed_yaml = yaml.safe_load(resp.read().decode('utf-8'))

raw_proxies = parsed_yaml.get("proxies", [])

landing_nodes = []
info_keywords = ["剩余流量", "到期", "时间", "官网", "重置", "套餐", "群", "建议", "通知", "公告", "更新", "维护", "规则", "说明", "导航", "客服", "traffic", "reset", "info", "subscribe"]

for p in raw_proxies:
    name = p.get("name", "")
    t = p.get("type", "")
    server = p.get("server", "")
    
    if not name or not t or not server:
        continue
    if any(k in name.lower() for k in info_keywords):
        continue
    if t == "http" and "info" in p.get("username", ""):
        continue
    landing_nodes.append(p)

print(f"Total parsed usable landing nodes: {len(landing_nodes)}")

# 2. Get Frontends from DB
c = sqlite3.connect('/data/proxybox/app/data/proxybox.db').cursor()
trojan_raw = json.loads(c.execute("SELECT raw_config FROM nodes WHERE type='trojan' AND original_name LIKE '%日本03%'").fetchone()[0])
trojan_raw['name'] = 'p-trojan-jp'
trojan_raw.pop('dialer-proxy', None)

hy2_raw = json.loads(c.execute("SELECT raw_config FROM nodes WHERE type='hysteria2' AND original_name LIKE '%香港01%'").fetchone()[0])
hy2_raw['name'] = 'p-hy2-hk'
hy2_raw.pop('dialer-proxy', None)

proxies_cfg = [trojan_raw, hy2_raw]

for idx, n in enumerate(landing_nodes):
    nid = f"n{idx}"
    
    # 1. Direct
    d = dict(n)
    d['name'] = f"d_{nid}"
    d.pop('dialer-proxy', None)
    proxies_cfg.append(d)
    
    # 2. Via Trojan
    t = dict(n)
    t['name'] = f"t_{nid}"
    t['dialer-proxy'] = 'p-trojan-jp'
    proxies_cfg.append(t)
    
    # 3. Via Hy2
    h = dict(n)
    h['name'] = f"h_{nid}"
    h['dialer-proxy'] = 'p-hy2-hk'
    proxies_cfg.append(h)

cfg = {
    "mode": "rule",
    "log-level": "warning",
    "external-controller": f"127.0.0.1:{PORT}",
    "secret": SECRET,
    "proxies": proxies_cfg,
    "rules": ["MATCH,DIRECT"]
}

cfg_path = os.path.join(WORK_DIR, "config.json")
with open(cfg_path, 'w', encoding='utf-8') as f:
    json.dump(cfg, f, indent=2)

# Config check
chk = subprocess.run([BIN_PATH, "-t", "-d", WORK_DIR, "-f", cfg_path], capture_output=True, text=True)
if chk.returncode != 0:
    print("Config validation failed:", chk.stderr)
    exit(1)

log_path = os.path.join(WORK_DIR, "mihomo.log")
log_f = open(log_path, "w", encoding='utf-8')
proc = subprocess.Popen([BIN_PATH, "-d", WORK_DIR, "-f", cfg_path], stdout=log_f, stderr=subprocess.STDOUT)
time.sleep(2)

def test_delay(proxy_name, timeout_ms=6000):
    enc_name = urllib.parse.quote(proxy_name)
    url = urllib.parse.quote("https://cp.cloudflare.com/generate_204")
    api_url = f"http://127.0.0.1:{PORT}/proxies/{enc_name}/delay?url={url}&timeout={timeout_ms}"
    req = urllib.request.Request(api_url, headers={"Authorization": f"Bearer {SECRET}"})
    try:
        with urllib.request.urlopen(req, timeout=timeout_ms/1000.0 + 1) as resp:
            data = json.loads(resp.read().decode('utf-8'))
            return data.get("delay"), None
    except urllib.error.HTTPError as e:
        body = e.read().decode('utf-8', errors='ignore')
        if "timeout" in body.lower():
            return None, "超时"
        return None, "握手失败"
    except Exception as e:
        err_str = str(e).lower()
        if "timed out" in err_str:
            return None, "超时"
        return None, "连接异常"

def test_single_node(idx_node):
    idx, n = idx_node
    nid = f"n{idx}"
    name = n.get("name")
    proto = n.get("type")
    
    d_ms, d_err = test_delay(f"d_{nid}")
    t_ms, t_err = test_delay(f"t_{nid}")
    h_ms, h_err = test_delay(f"h_{nid}")
    
    return {
        "index": idx,
        "name": name,
        "type": proto,
        "direct": f"{d_ms}ms" if d_ms else d_err,
        "trojan": f"{t_ms}ms" if t_ms else t_err,
        "hy2": f"{h_ms}ms" if h_ms else h_err
    }

print("Testing all nodes concurrently with 6 workers...")
with ThreadPoolExecutor(max_workers=6) as ex:
    results = list(ex.map(test_single_node, enumerate(landing_nodes)))

results.sort(key=lambda x: x['index'])

proc.terminate()
proc.wait()
log_f.close()

# Print markdown table
print("\n" + "="*80)
print("### 全量落地矩阵测试结果表")
print("="*80)
print("| 节点名称 | 协议 | 直连延迟 | 经 Trojan (日本03) | 经 Hy2 (香港01) | 状态分类 |")
print("|---|---|---|---|---|---|")
for r in results:
    status = "✅ 链式成功" if "ms" in r['trojan'] and "ms" in r['hy2'] else ("⚠️ 部分成功" if "ms" in r['trojan'] or "ms" in r['hy2'] else "❌ 失败")
    print(f"| {r['name']} | `{r['type']}` | {r['direct']} | {r['trojan']} | {r['hy2']} | {status} |")

# Check missing regions compared to old CNIX list
old_cnix_names = [
    "澳门（动态家庭）", "香港WiFicall（家庭）", "台湾（家庭）", "台湾（动态家庭）",
    "菲律宾（动态家宽）", "新加坡（动态家庭）", "泰国（动态家庭）", "缅甸（家庭）",
    "韩国（动态家庭）", "日本家庭（测试）", "日本（动态家庭）", "印尼（家庭）",
    "美国（轮换家庭）", "英国（轮换家庭）", "新西兰家庭", "乌克兰（轮换家庭）",
    "土耳其（轮换家庭）", "捷克（家庭）", "德国（轮换家庭）", "法国（轮换家庭）",
    "瑞士（机房）", "西班牙WiFicall（轮换家庭）", "以色列（动态家庭）",
    "巴基斯坦WiFicall（动态家庭）", "刚果WiFicall（动态家庭）"
]

current_names = [r['name'] for r in results]
missing = []
for old in old_cnix_names:
    norm_old = re.sub(r'\(.*?\)|（.*?）|\s+', '', old)
    found = any(norm_old in re.sub(r'\(.*?\)|（.*?）|\s+', '', c) for c in current_names)
    if not found:
        missing.append(old)

print("\n" + "="*80)
print("### 原生订阅地区缺失对比清单")
print("="*80)
if missing:
    print("在原生订阅中未发现的旧 CNIX 节点地区:")
    for m in missing:
        print(f"- {m}")
else:
    print("旧 CNIX 订阅的所有 25 个地区在原生订阅中全部存在！")
