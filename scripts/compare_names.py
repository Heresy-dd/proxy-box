import urllib.request
import ssl
import json
import re

ctx = ssl.create_default_context()
ctx.check_hostname = False
ctx.verify_mode = ssl.CERT_NONE

# 1. Fetch with cnix=1
req1 = urllib.request.Request("https://sub.qianlink.link/customer-sub/LGPbtwU1i9SZGVtbGIBa-5UyfzZuXcj7/qianlink.heresy?cnix=1", headers={"User-Agent": "clash.meta"})
with urllib.request.urlopen(req1, context=ctx) as r:
    b1 = r.read().decode('utf-8')

# 2. Fetch without cnix
req2 = urllib.request.Request("https://sub.qianlink.link/customer-sub/LGPbtwU1i9SZGVtbGIBa-5UyfzZuXcj7/qianlink.heresy", headers={"User-Agent": "clash.meta"})
with urllib.request.urlopen(req2, context=ctx) as r:
    b2 = r.read().decode('utf-8')

def get_names(text):
    names = []
    for line in text.splitlines():
        if "  - name:" in line:
            m = re.search(r'name:\s*(.+)', line)
            if m:
                names.append(m.group(1).strip().strip('"').strip("'"))
    return names

names1 = get_names(b1)
names2 = get_names(b2)

print("=== Names with cnix=1 (count=%d) ===" % len(names1))
for n in names1[:15]:
    print(" ", n)

print("\n=== Names without cnix (count=%d) ===" % len(names2))
for n in names2[:15]:
    print(" ", n)

print("\n=== Matching comparison ===")
# Check how many names match exactly or fuzzy
for n2 in names2:
    exact = [n1 for n1 in names1 if n1 == n2]
    # Check if normalized without "CNIX专线 · " matches
    norm1 = [n1 for n1 in names1 if n1.replace("CNIX专线 · ", "").split()[0] in n2]
    print(f"Native: [{n2}] -> Exact match: {exact} | Substring match: {norm1}")
