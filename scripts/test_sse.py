import time
import requests
import sys

def test_stream(url, duration=15):
    print(f"Testing SSE stream on: {url}")
    t0 = time.time()
    try:
        resp = requests.get(url, stream=True, timeout=20)
        print(f"Status: {resp.status_code}")
        print("Headers:")
        for k, v in resp.headers.items():
            if k.lower() in ('content-type', 'cache-control', 'transfer-encoding', 'connection', 'x-accel-buffering'):
                print(f"  {k}: {v}")
        
        event_count = 0
        for line in resp.iter_lines():
            now = time.time() - t0
            if line:
                decoded = line.decode('utf-8', errors='ignore')
                print(f"[{now:.2f}s] {decoded}")
                event_count += 1
            if time.time() - t0 > duration:
                print("Duration reached, stopping.")
                break
        print(f"Completed: received {event_count} lines in {time.time() - t0:.2f}s")
    except Exception as e:
        print(f"Stream error after {time.time() - t0:.2f}s: {e}")

if __name__ == '__main__':
    mode = sys.argv[1] if len(sys.argv) > 1 else 'local'
    if mode == 'local':
        url = "http://127.0.0.1:8080/api/speedtest/stream?front_id=&url=https://cp.cloudflare.com/generate_204&timeout=5&concurrency=5"
    else:
        url = "https://8080-agq4t5bbhw7w7ws7m352j5y6py.instabox.tech/api/speedtest/stream?front_id=&url=https://cp.cloudflare.com/generate_204&timeout=5&concurrency=5"
    test_stream(url)
