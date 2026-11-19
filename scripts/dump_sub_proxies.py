import urllib.request
import ssl

url = "https://sub.qianlink.link/customer-sub/LGPbtwU1i9SZGVtbGIBa-5UyfzZuXcj7/qianlink.heresy?cnix=1"
req = urllib.request.Request(url, headers={"User-Agent": "clash.meta"})
ctx = ssl.create_default_context()
ctx.check_hostname = False
ctx.verify_mode = ssl.CERT_NONE

with urllib.request.urlopen(req, context=ctx, timeout=10) as resp:
    body = resp.read().decode('utf-8', errors='ignore')

lines = body.splitlines()
in_proxies = False
proxy_lines = []
for line in lines:
    if line.startswith("proxies:"):
        in_proxies = True
        continue
    if in_proxies:
        if line.startswith("proxy-groups:") or line.startswith("rules:"):
            break
        proxy_lines.append(line)

print("\n".join(proxy_lines[:50]))
