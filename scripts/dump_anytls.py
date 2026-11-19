import urllib.request
import ssl
import json

url1 = "https://sub.qianlink.link/customer-sub/LGPbtwU1i9SZGVtbGIBa-5UyfzZuXcj7/qianlink.heresy"
req = urllib.request.Request(url1, headers={"User-Agent": "clash.meta"})
ctx = ssl.create_default_context()
ctx.check_hostname = False
ctx.verify_mode = ssl.CERT_NONE

with urllib.request.urlopen(req, context=ctx, timeout=10) as resp:
    body = resp.read().decode('utf-8', errors='ignore')

# Extract one proxy
lines = body.splitlines()
in_proxies = False
proxy_block = []
for line in lines:
    if line.startswith("proxies:"):
        in_proxies = True
        continue
    if in_proxies:
        if "澳门（动态家庭）" in line:
            proxy_block.append(line)
        elif proxy_block and line.startswith("  - name:"):
            break
        elif proxy_block:
            proxy_block.append(line)

print("\n".join(proxy_block))
