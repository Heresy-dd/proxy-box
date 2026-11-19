import sqlite3
import json

db_path = '/data/proxybox/app/data/proxybox.db'
conn = sqlite3.connect(db_path)
c = conn.cursor()

print("=== SUBSCRIPTIONS ===")
for r in c.execute("SELECT id, name, url, total_bytes, expire_timestamp FROM subscriptions").fetchall():
    print(r)

print("\n=== SAMPLE NODES BY TYPE ===")
types = set(r[0] for r in c.execute("SELECT DISTINCT type FROM nodes").fetchall())
print("Distinct types:", types)

for t in sorted(types):
    nodes = c.execute("SELECT id, name, type, server, port, pool, raw_config FROM nodes WHERE type=? LIMIT 2", (t,)).fetchall()
    for n in nodes:
        print(f"[{t}] id={n[0]} name={n[1]} pool={n[5]}")
        raw = json.loads(n[6])
        print("   raw keys:", list(raw.keys()))
        print("   details:", {k: v for k, v in raw.items() if k not in ('password', 'uuid')})

print("\n=== CHAINS ===")
for r in c.execute("SELECT id, name, frontend_node_id, landing_node_id, listen_port, username, password, enabled, status, outbound_ip, outbound_country FROM chains").fetchall():
    print(r)
