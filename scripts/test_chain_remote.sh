#!/bin/bash
set -e
BIN=/tmp/mihomo-test/mihomo-linux-amd64-v1.19.34
$BIN -d /tmp/mihomo-test -f /tmp/mihomo-test/real-chain-test.yaml > /tmp/mihomo-test/mihomo.log 2>&1 &
PID=$!
echo "Mihomo started with PID: $PID"
sleep 2

echo "--- 1. External Controller Health ---"
curl -s -H "Authorization: Bearer chainsecret" http://127.0.0.1:9095/version
echo ""

echo "--- 2. Delay Test for front-vless-sg ---"
curl -s -H "Authorization: Bearer chainsecret" "http://127.0.0.1:9095/proxies/front-vless-sg/delay?url=https%3A%2F%2Fcp.cloudflare.com%2Fgenerate_204&timeout=5000"
echo ""

echo "--- 3. Delay Test for direct-macau ---"
curl -s -H "Authorization: Bearer chainsecret" "http://127.0.0.1:9095/proxies/direct-macau/delay?url=https%3A%2F%2Fcp.cloudflare.com%2Fgenerate_204&timeout=5000"
echo ""

echo "--- 4. Delay Test for chain-macau (dialer-proxy: front-vless-sg) ---"
curl -s -H "Authorization: Bearer chainsecret" "http://127.0.0.1:9095/proxies/chain-macau/delay?url=https%3A%2F%2Fcp.cloudflare.com%2Fgenerate_204&timeout=5000"
echo ""

echo "--- 5. Outbound IP via Port 9002 (direct-macau) ---"
curl -s -m 8 -x http://testuser:testpassword@127.0.0.1:9002 https://ipinfo.io/json || echo "Port 9002 curl failed"
echo ""

echo "--- 6. Outbound IP via Port 9001 (chain-macau dialer-proxy) ---"
curl -s -m 8 -x http://testuser:testpassword@127.0.0.1:9001 https://ipinfo.io/json || echo "Port 9001 curl failed"
echo ""

kill -9 $PID 2>/dev/null || true
echo "Test finished."
