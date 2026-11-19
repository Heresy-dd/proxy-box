#!/bin/bash
BIN=/tmp/mihomo-test/mihomo-linux-amd64-v1.19.34
pkill -9 -e mihomo || true
sleep 1

cat << 'EOF' > /tmp/mihomo-test/chain_hy2_trojan.yaml
mode: rule
log-level: info
external-controller: 127.0.0.1:9095
secret: 'chainsecret'

proxies:
  - name: front-hy2-hk
    type: hysteria2
    server: 42.200.231.232
    port: 443
    password: 042f5fc2-0eda-49ab-abb6-3254bf296244
    udp: true
    skip-cert-verify: true
    sni: www.bing.com

  - name: landing-trojan-jp
    type: trojan
    server: 35.77.107.225
    port: 443
    password: 042f5fc2-0eda-49ab-abb6-3254bf296244
    udp: true
    sni: www.bing.com
    skip-cert-verify: true
    dialer-proxy: front-hy2-hk

listeners:
  - name: in-9001
    type: mixed
    listen: 127.0.0.1
    port: 9001
    proxy: landing-trojan-jp
    udp: true
    users:
      - username: testuser
        password: testpassword

rules:
  - MATCH,DIRECT
EOF

$BIN -d /tmp/mihomo-test -f /tmp/mihomo-test/chain_hy2_trojan.yaml > /tmp/mihomo-test/hy2_trojan.log 2>&1 &
PID=$!
sleep 2

echo "=== Delay Tests ==="
echo -n "Front (HK Hy2): "
curl -s -H 'Authorization: Bearer chainsecret' 'http://127.0.0.1:9095/proxies/front-hy2-hk/delay?url=https%3A%2F%2Fcp.cloudflare.com%2Fgenerate_204&timeout=5000'
echo ""

echo -n "Chain (HK Hy2 -> JP Trojan): "
curl -s -H 'Authorization: Bearer chainsecret' 'http://127.0.0.1:9095/proxies/landing-trojan-jp/delay?url=https%3A%2F%2Fcp.cloudflare.com%2Fgenerate_204&timeout=8000'
echo ""

echo "=== Outbound IP via Mixed Port 9001 HTTP ==="
curl -s -m 8 -x http://testuser:testpassword@127.0.0.1:9001 https://api.ipify.org?format=json || echo "Port 9001 failed"
echo ""

pkill -9 -e mihomo || true
