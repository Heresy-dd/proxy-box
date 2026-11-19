#!/bin/bash
BIN=/tmp/mihomo-test/mihomo-linux-amd64-v1.19.34

# Extract first 5 proxies from sub1.yaml (skipping info nodes)
echo "=== Sub1 Proxies ==="
grep -E "name:|type:|server:|port:" /tmp/mihomo-test/sub1.yaml | grep -v "剩余流量" | grep -v "官网" | grep -v "套餐" | grep -v "建议" | head -n 30
