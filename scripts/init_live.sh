#!/bin/bash
# 1. Setup admin password
SETUP_RES=$(curl -s -X POST http://127.0.0.1:8080/api/auth/setup -H 'Content-Type: application/json' -d '{"password":"proxybox123"}')
echo "Setup response: $SETUP_RES"

# 2. Login
TOKEN=$(curl -s -X POST http://127.0.0.1:8080/api/auth/login -H 'Content-Type: application/json' -d '{"password":"proxybox123"}' | grep -o '"token":"[^"]*' | cut -d'"' -f4)
echo "Obtained Token: $TOKEN"

# 3. Add Subscription 1 (前置机场)
echo "Adding Subscription 1..."
curl -s -X POST http://127.0.0.1:8080/api/subscriptions \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"name":"宝可梦·前置机场","url":"https://link123.52pokemon66.cc/api/v1/client/subscribe?token=5a36317c093eea5cca0b6cfb05f4a6db","update_interval":60}'

echo ""
echo "Adding Subscription 2..."
curl -s -X POST http://127.0.0.1:8080/api/subscriptions \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"name":"千链·住宅专线","url":"https://sub.qianlink.link/customer-sub/LGPbtwU1i9SZGVtbGIBa-5UyfzZuXcj7/qianlink.heresy?cnix=1","update_interval":120}'

echo ""
echo "Listing Subscriptions..."
curl -s http://127.0.0.1:8080/api/subscriptions -H "Authorization: Bearer $TOKEN"
echo ""

echo "Querying Pools Summary..."
curl -s http://127.0.0.1:8080/api/pools -H "Authorization: Bearer $TOKEN" | grep -o '"pool":"[^"]*' | sort | uniq -c
