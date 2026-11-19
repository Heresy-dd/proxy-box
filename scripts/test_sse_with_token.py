import urllib.request
import json
import time

def login(base_url, password="proxybox123"):
    req = urllib.request.Request(f"{base_url}/api/auth/login", 
                                 data=json.dumps({"password": password}).encode('utf-8'),
                                 headers={"Content-Type": "application/json"})
    try:
        with urllib.request.urlopen(req, timeout=5) as resp:
            data = json.loads(resp.read().decode('utf-8'))
            return data.get("token")
    except Exception as e:
        print(f"Login to {base_url} failed:", e)
        return None

def test_sse(base_url, token):
    url = f"{base_url}/api/speedtest/stream?token={token}&timeout=3&concurrency=2"
    print(f"\n--- Testing SSE: {url} ---")
    t0 = time.time()
    req = urllib.request.Request(url)
    try:
        with urllib.request.urlopen(req, timeout=15) as resp:
            print("Status:", resp.status)
            print("Headers:")
            for k, v in resp.headers.items():
                if k.lower() in ('content-type', 'cache-control', 'transfer-encoding', 'connection', 'x-accel-buffering'):
                    print(f"  {k}: {v}")
            # Read first few lines
            count = 0
            while True:
                line = resp.readline()
                if not line:
                    print("Stream closed by server.")
                    break
                elapsed = time.time() - t0
                print(f"[{elapsed:.2f}s] {line.decode('utf-8', errors='ignore').strip()}")
                count += 1
                if count >= 8 or elapsed > 10:
                    break
    except Exception as e:
        print(f"Stream error after {time.time()-t0:.2f}s: {e}")

local_url = "http://127.0.0.1:8080"
token = login(local_url)
print("Got token:", token[:20] if token else "None")
if token:
    print("\n=== Testing Localhost SSE ===")
    test_sse(local_url, token)
    
    print("\n=== Testing Remote Gateway SSE ===")
    remote_url = "https://8080-agq4t5bbhw7w7ws7m352j5y6py.instabox.tech"
    test_sse(remote_url, token)
