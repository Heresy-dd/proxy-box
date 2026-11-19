import urllib.request
import ssl

url1 = "https://sub.qianlink.link/customer-sub/LGPbtwU1i9SZGVtbGIBa-5UyfzZuXcj7/qianlink.heresy"
req = urllib.request.Request(url1, headers={"User-Agent": "clash.meta"})
ctx = ssl.create_default_context()
ctx.check_hostname = False
ctx.verify_mode = ssl.CERT_NONE

try:
    with urllib.request.urlopen(req, context=ctx, timeout=10) as resp:
        print("Status (no cnix):", resp.status)
        body = resp.read().decode('utf-8', errors='ignore')
        print("Body preview without cnix:")
        for line in body.splitlines()[:30]:
            print(" ", line)
except Exception as e:
    print("Fetch without cnix failed:", e)
