import urllib.request
import ssl

url1 = "https://sub.qianlink.link/customer-sub/LGPbtwU1i9SZGVtbGIBa-5UyfzZuXcj7/qianlink.heresy"
req = urllib.request.Request(url1, headers={"User-Agent": "clash.meta"})
ctx = ssl.create_default_context()
ctx.check_hostname = False
ctx.verify_mode = ssl.CERT_NONE

with urllib.request.urlopen(req, context=ctx, timeout=10) as resp:
    body = resp.read().decode('utf-8', errors='ignore')

in_proxies = False
for line in body.splitlines():
    if line.startswith("proxies:"):
        in_proxies = True
        continue
    if in_proxies:
        if line.startswith("proxy-groups:"):
            break
        if "name:" in line or "server:" in line or "type:" in line or "port:" in line:
            print(line)
