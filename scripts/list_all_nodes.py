import sqlite3
import json

conn = sqlite3.connect('/data/proxybox/app/data/proxybox.db')
c = conn.cursor()

nodes = c.execute("SELECT id, name, type, server, port, pool, raw_config, sub_name FROM nodes").fetchall()
print(f"Total nodes: {len(nodes)}")
for n in nodes:
    raw = json.loads(n[6])
    print(f"Sub=[{n[7]}] ID={n[0]} Name={n[1]} Type={n[2]} Server={n[3]}:{n[4]} UDP={raw.get('udp', 'unset')}")
