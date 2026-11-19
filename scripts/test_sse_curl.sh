#!/bin/bash
echo "=== 1. Testing SSE on Localhost (127.0.0.1:8080) ==="
curl -N -v -m 10 "http://127.0.0.1:8080/api/speedtest/stream?front_id=&url=https://cp.cloudflare.com/generate_204&timeout=3&concurrency=2" 2>&1 | head -n 30

echo -e "\n=== 2. Testing SSE on Public Gateway (https://8080-agq4t5bbhw7w7ws7m352j5y6py.instabox.tech) ==="
curl -N -v -m 10 "https://8080-agq4t5bbhw7w7ws7m352j5y6py.instabox.tech/api/speedtest/stream?front_id=&url=https://cp.cloudflare.com/generate_204&timeout=3&concurrency=2" 2>&1 | head -n 30
