#!/bin/bash
TOKEN=$(curl -s -X POST http://127.0.0.1:8080/api/auth/login -H 'Content-Type: application/json' -d '{"password":"proxybox123"}' | grep -o '"token":"[^"]*' | cut -d'"' -f4)

# 1. Get an ID from Pokemon sub for frontend
FRONT_ID=$(curl -s http://127.0.0.1:8080/api/pools -H "Authorization: Bearer $TOKEN" | python3 -c '
import sys, json
pools = json.load(sys.stdin)
for n in pools.get("unassigned", []):
    if "香港" in n.get("name", ""):
        print(n["id"])
        break
')

# 2. Get an ID from Qianlink sub for landing
LANDING_ID=$(curl -s http://127.0.0.1:8080/api/pools -H "Authorization: Bearer $TOKEN" | python3 -c '
import sys, json
pools = json.load(sys.stdin)
for n in pools.get("unassigned", []):
    if "澳门" in n.get("name", "") or "香港WiFi" in n.get("name", ""):
        print(n["id"])
        break
')

echo "Selected Frontend Node ID: $FRONT_ID"
echo "Selected Landing Node ID: $LANDING_ID"

# 3. Move nodes to respective pools
echo "Moving nodes to pools..."
curl -s -X POST http://127.0.0.1:8080/api/nodes/move -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d "{\"node_ids\":[\"$FRONT_ID\"],\"target_pool\":\"frontend\"}"
echo ""
curl -s -X POST http://127.0.0.1:8080/api/nodes/move -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d "{\"node_ids\":[\"$LANDING_ID\"],\"target_pool\":\"landing\"}"
echo ""

# 4. Create chain
echo "Creating chain on port 9001..."
CHAIN_RES=$(curl -s -X POST http://127.0.0.1:8080/api/chains -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d "{
  \"name\": \"香港前置 -> 澳门家庭住宅\",
  \"frontend_node_id\": \"$FRONT_ID\",
  \"landing_node_id\": \"$LANDING_ID\",
  \"listen_port\": 9001,
  \"enabled\": true
}")
echo "Chain created: $CHAIN_RES"

# 5. Check staged diff
echo ""
echo "Staged Diff:"
curl -s http://127.0.0.1:8080/api/apply/diff -H "Authorization: Bearer $TOKEN"
echo ""

# 6. Apply configuration!
echo ""
echo "Applying production configuration..."
APPLY_RES=$(curl -s -X POST http://127.0.0.1:8080/api/apply -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"description":"初始化建立 9001 端口链式代理"}')
echo "Apply result: $APPLY_RES"

# 7. Check listening ports
echo ""
echo "Checking system listening ports for 9001:"
ss -tulpn | grep 9001 || true
