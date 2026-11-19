import urllib.request
import ssl

url = "https://sub.qianlink.link/customer-sub/LGPbtwU1i9SZGVtbGIBa-5UyfzZuXcj7/qianlink.heresy?cnix=1"
req = urllib.request.Request(url, headers={"User-Agent": "clash.meta"})
ctx = ssl.create_default_context()
ctx.check_hostname = False
ctx.verify_mode = ssl.CERT_NONE

try:
    with urllib.request.urlopen(req, context=ctx, timeout=10) as resp:
        print("Status:", resp.status)
        print("Headers:")
        for k, v in resp.headers.items():
            print(f"  {k}: {v}")
        body = resp.read().decode('utf-8', errors='ignore')
        print("Body preview (first 500 chars):")
        print(body[:500])
except Exception as e:
    print("Fetch failed:", e)
