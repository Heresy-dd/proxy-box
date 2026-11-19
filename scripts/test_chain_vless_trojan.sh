#!/bin/bash
BIN=/tmp/mihomo-test/mihomo-linux-amd64-v1.19.34

cat << 'EOF' > /tmp/mihomo-test/chain_vless_trojan.yaml
mode: rule
log-level: info
external-controller: 127.0.0.1:9095
secret: 'chainsecret'

proxies:
  - name: front-vless-sg
    type: vless
    server: 54.255.163.212
    port: 443
    uuid: 042f5fc2-0eda-49ab-abb6-3254bf296244
    udp: true
    flow: xtls-rprx-vision
    tls: true
    skip-cert-verify: true
    client-fingerprint: chrome
    servername: www.shopify.com

  - name: direct-trojan-jp
    type: trojan
    server: 35.77.107.225
    port: 443
    password: 042f5fc2-0eda-49ab-abb6-3254bf296244
    udp: true
    sni: www.bing.com
    skip-cert-verify: true

  - name: chain-sg-to-jp-trojan
    type: trojan
    server: 35.77.107.225
    port: 443
    password: 042f5fc2-0eda-49ab-abb6-3254bf296244
    udp: true
    sni: www.bing.com
    skip-cert-verify: true
    dialer-proxy: front-vless-sg

listeners:
  - name: in-9001
    type: mixed
    listen: 127.0.0.1
    port: 9001
    proxy: chain-sg-to-jp-trojan
    udp: true
    users:
      - username: testuser
        password: testpassword

  - name: in-9002
    type: mixed
    listen: 127.0.0.1
    port: 9002
    proxy: direct-trojan-jp
    udp: true
    users:
      - username: testuser
        password: testpassword

rules:
  - MATCH,DIRECT
EOF

$BIN -d /tmp/mihomo-test -f /tmp/mihomo-test/chain_vless_trojan.yaml > /tmp/mihomo-test/trojan.log 2>&1 &
PID=$!
sleep 2

echo "=== Delay Tests ==="
echo -n "Front (SG Vless): "
curl -s -H 'Authorization: Bearer chainsecret' 'http://127.0.0.1:9095/proxies/front-vless-sg/delay?url=https%3A%2F%2Fcp.cloudflare.com%2Fgenerate_204&timeout=5000'
echo ""

echo -n "Direct (JP Trojan): "
curl -s -H 'Authorization: Bearer chainsecret' 'http://127.0.0.1:9095/proxies/direct-trojan-jp/delay?url=https%3A%2F%2Fcp.cloudflare.com%2Fgenerate_204&timeout=5000'
echo ""

echo -n "Chain (SG Vless -> JP Trojan): "
curl -s -H 'Authorization: Bearer chainsecret' 'http://127.0.0.1:9095/proxies/chain-sg-to-jp-trojan/delay?url=https%3A%2F%2Fcp.cloudflare.com%2Fgenerate_204&timeout=8000'
echo ""

echo "=== Egress IP via SOCKS5 (Port 9001: Chain SG -> JP) ==="
curl -s -m 8 -x socks5://testuser:testpassword@127.0.0.1:9001 https://api.ipify.org?format=json || echo "Failed"
echo ""

echo "=== Egress IP via HTTP (Port 9001: Mixed Listener) ==="
curl -s -m 8 -x http://testuser:testpassword@127.0.0.1:9001 https://api.ipify.org?format=json || echo "Failed"
echo ""

kill -9 $PID 2>/dev/null || true
