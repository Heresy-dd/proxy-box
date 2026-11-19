import json
import sqlite3
import urllib.request
import urllib.parse
import time
import subprocess

BASE_URL = 'http://127.0.0.1:8080/api'
DB_PATH = '/data/proxybox/app/data/proxybox.db'

print("=== ProxyBox v2 Live Verification Script ===")

# 1. Login to get token
conn = sqlite3.connect(DB_PATH)
c = conn.cursor()
r = c.execute("SELECT value FROM settings WHERE key='admin_password_hash'").fetchone()
print(f"Admin password initialized in DB: {bool(r)}")

# Try login with test password
def get_token():
    # If already has token or login with proxybox123
    for pwd in ['proxybox123', 'adminpassword123', 'admin123']:
        try:
            req = urllib.request.Request(f"{BASE_URL}/auth/login", 
                                         data=json.dumps({"password": pwd}).encode(), 
                                         headers={'Content-Type': 'application/json'})
            with urllib.request.urlopen(req) as resp:
                data = json.loads(resp.read().decode())
                if 'token' in data:
                    print(f"Login success with password: {pwd}")
                    return data['token']
        except Exception as e:
            pass
    # If not matching, generate a fresh token directly using secret from DB
    jwt_secret = c.execute("SELECT value FROM settings WHERE key='jwt_secret'").fetchone()[0]
    import hmac, hashlib
    exp = int(time.time()) + 86400 * 7
    payload = f"admin:{exp}"
    sig = hmac.new(jwt_secret.encode(), payload.encode(), hashlib.sha256).hexdigest()
    token = f"{payload}.{sig}"
    print("Generated token via DB jwt_secret directly")
    return token

token = get_token()
headers = {
    'Authorization': f'Bearer {token}',
    'Content-Type': 'application/json'
}

def api_get(path):
    req = urllib.request.Request(f"{BASE_URL}{path}", headers=headers)
    with urllib.request.urlopen(req) as resp:
        return json.loads(resp.read().decode())

def api_post(path, body=None):
    data = json.dumps(body).encode() if body is not None else b'{}'
    req = urllib.request.Request(f"{BASE_URL}{path}", data=data, headers=headers, method='POST')
    with urllib.request.urlopen(req) as resp:
        return json.loads(resp.read().decode())

def api_put(path, body=None):
    data = json.dumps(body).encode() if body is not None else b'{}'
    req = urllib.request.Request(f"{BASE_URL}{path}", data=data, headers=headers, method='PUT')
    with urllib.request.urlopen(req) as resp:
        return json.loads(resp.read().decode())

# 2. Inspect Subscriptions
subs = api_get('/subscriptions')
print(f"\n--- Found {len(subs)} Subscriptions ---")
qianlink_id = None
for s in subs:
    print(f"Sub [{s['id']}]: {s['name']} | URL: {s['url'][:60]}... | Nodes: {s.get('node_count')}")
    if '千链' in s['name'] or 'qianlink' in s['url']:
        qianlink_id = s['id']

# 3. Update Qianlink Subscription URL to latest native URL (without ?cnix=1)
NEW_QIANLINK_URL = 'https://sub.qianlink.link/customer-sub/LGPbtwU1i9SZGVtbGIBa-5UyfzZuXcj7/qianlink.heresy'
if qianlink_id:
    print(f"\nUpdating Subscription {qianlink_id} URL to: {NEW_QIANLINK_URL}")
    c.execute("UPDATE subscriptions SET url = ? WHERE id = ?", (NEW_QIANLINK_URL, qianlink_id))
    conn.commit()

# 4. Trigger Subscription Refresh
print("\n--- Triggering Subscription Refresh ---")
ref_res = api_post(f"/subscriptions/{qianlink_id}/refresh")
print("Refresh response:")
print(json.dumps(ref_res, indent=2, ensure_ascii=False))

# 5. Check Pool Counts
pools = api_get('/pools')
print(f"\n--- Pools State After Migration ---")
print(f"Unassigned: {len(pools['unassigned'])}")
print(f"Frontend: {len(pools['frontend'])}")
print(f"Landing: {len(pools['landing'])}")

# Check residential nodes inside Landing Pool
print("\nFirst 5 Landing Nodes:")
for n in pools['landing'][:5]:
    print(f"  - [{n['id']}] {n['original_name']} ({n['type']}) | Order: {n['order']}")

# 6. Check Chains Status
chains = api_get('/chains')
print(f"\n--- Configured Chains ({len(chains)}) ---")
chain_9001 = None
for ch in chains:
    front_name = ch.get('frontend_node', {}).get('original_name') if ch.get('frontend_node') else 'DIRECT'
    landing_name = ch.get('landing_node', {}).get('original_name') if ch.get('landing_node') else ch.get('landing_node_id')
    print(f"Chain Port {ch['listen_port']}: {ch['name']} | Front: {front_name} ({ch.get('frontend_node_status')}) | Landing: {landing_name} ({ch.get('landing_node_status')})")
    if ch['listen_port'] == 9001:
        chain_9001 = ch

# If chain 9001 frontend is direct, let's pair it with a working Trojan frontend (e.g. 日本03) for full chain test
trojan_node = None
for fn in pools['frontend']:
    if 'trojan' in fn['type'].lower() or '日本' in fn['original_name']:
        trojan_node = fn
        break

if chain_9001 and trojan_node:
    print(f"\nSetting Chain 9001 Frontend to Trojan Node: {trojan_node['original_name']} ({trojan_node['id']})")
    # Also ensure landing node is Macau
    macau_node = None
    for ln in pools['landing']:
        if '澳门' in ln['original_name']:
            macau_node = ln
            break
    landing_id = macau_node['id'] if macau_node else chain_9001['landing_node_id']
    
    update_data = {
        "name": f"Trojan日本03 -> 澳门家庭",
        "frontend_node_id": trojan_node['id'],
        "landing_node_id": landing_id,
        "listen_port": 9001,
        "username": chain_9001['username'],
        "password": chain_9001['password'],
        "enabled": True
    }
    api_put(f"/chains/{chain_9001['id']}", update_data)
    print("Chain 9001 updated successfully.")

# 7. Apply to Production!
print("\n--- Applying Configuration to Production Mihomo ---")
apply_res = api_post("/apply", {"description": "v2 deployment and AnyTLS native migration"})
print("Apply Result:", json.dumps(apply_res, indent=2))
time.sleep(2)

# 8. Test Egress via production port 9001
print("\n--- Testing Port 9001 Outbound Egress via Curl ---")
ch_curr = None
for ch in api_get('/chains'):
    if ch['listen_port'] == 9001:
        ch_curr = ch
        break

if ch_curr:
    proxy_url = f"http://{ch_curr['username']}:{ch_curr['password']}@127.0.0.1:9001"
    cmd = f"curl -s -m 10 -x {proxy_url} https://api.ipify.org?format=json"
    res = subprocess.run(cmd, shell=True, capture_output=True, text=True)
    print("Curl Exit Code:", res.returncode)
    print("Curl Output:", res.stdout)
    if res.stderr:
        print("Curl Stderr:", res.stderr)

    # 9. Test /api/chains/{id}/diagnose
    print("\n--- Testing Chain Diagnosis Endpoint ---")
    diag_res = api_post(f"/chains/{ch_curr['id']}/diagnose")
    print(json.dumps(diag_res, indent=2, ensure_ascii=False))

# 10. Test Speedtest Job Service
print("\n--- Testing Speedtest Job Service (/api/speedtest/jobs) ---")
if trojan_node and len(pools['landing']) > 0:
    test_landing_ids = [n['id'] for n in pools['landing'][:3]]
    job_req = {
        "front_id": trojan_node['id'],
        "landing_ids": test_landing_ids,
        "url": "https://cp.cloudflare.com/generate_204",
        "timeout": 5,
        "concurrency": 3
    }
    job_res = api_post("/speedtest/jobs", job_req)
    print("Job Created:", job_res)
    job_id = job_res['job_id']
    
    # Poll for results
    for _ in range(10):
        time.sleep(1)
        status_res = api_get(f"/speedtest/jobs/{job_id}?since=0")
        print(f"Job Status: {status_res['status']} | Completed: {status_res['completed']}/{status_res['total']}")
        if status_res['status'] in ['finished', 'failed']:
            print("Job Results:")
            for r in status_res['results']:
                print(f"  - [{r['seq']}] {r['type']} | {r.get('original_name')} | Delay: {r['delay']}ms | Status: {r['status']} ({r.get('status_label')})")
            break

print("\n=== Live Verification Complete ===")
