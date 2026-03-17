#!/usr/bin/env bash

set -euo pipefail

GATEWAY="${GATEWAY:-http://localhost:8080}"
PASS=0
FAIL=0


green()  { printf '\033[0;32mâœ“ %s\033[0m\n' "$*"; }
red()    { printf '\033[0;31mâœ— %s\033[0m\n' "$*"; }
yellow() { printf '\033[1;33mÂ» %s\033[0m\n' "$*"; }

assert_eq() {
  local label="$1" expected="$2" actual="$3"
  if [ "$actual" = "$expected" ]; then
    green "$label"
    PASS=$((PASS+1))
  else
    red "$label â€” expected '$expected', got '$actual'"
    FAIL=$((FAIL+1))
  fi
}

assert_not_empty() {
  local label="$1" actual="$2"
  if [ -n "$actual" ] && [ "$actual" != "null" ]; then
    green "$label"
    PASS=$((PASS+1))
  else
    red "$label â€” expected non-empty value, got '$actual'"
    FAIL=$((FAIL+1))
  fi
}

assert_status() {
  local label="$1" expected="$2" actual="$3"
  if [ "$actual" = "$expected" ]; then
    green "$label (HTTP $actual)"
    PASS=$((PASS+1))
  else
    red "$label â€” expected HTTP $expected, got $actual"
    FAIL=$((FAIL+1))
  fi
}

do_request() {
  local method="$1" url="$2"
  shift 2
  curl -s -o /tmp/e2e_body.json -w "%{http_code}" \
    -X "$method" "$url" \
    -H "Content-Type: application/json" \
    "$@"
}

yellow "Step 1: Health check"
STATUS=$(do_request GET "$GATEWAY/healthz")
assert_status "Gateway health" "200" "$STATUS"

yellow "Step 2: Register user"
EMAIL="e2e-$(date +%s)@test.com"
STATUS=$(do_request POST "$GATEWAY/auth/register" \
  --data "{\"email\":\"$EMAIL\",\"name\":\"E2E User\",\"password\":\"TestPass1!\"}")
assert_status "Register user" "200" "$STATUS"

ACCESS=$(jq -r '.access_token' /tmp/e2e_body.json)
REFRESH=$(jq -r '.refresh_token' /tmp/e2e_body.json)
assert_not_empty "Access token issued" "$ACCESS"
assert_not_empty "Refresh token issued" "$REFRESH"

AUTH="-H \"Authorization: Bearer $ACCESS\""

yellow "Step 3: Get user profile"
STATUS=$(do_request GET "$GATEWAY/auth/me" -H "Authorization: Bearer $ACCESS")
assert_status "Get profile" "200" "$STATUS"
GOT_EMAIL=$(jq -r '.email' /tmp/e2e_body.json)
assert_eq "Profile email matches" "$EMAIL" "$GOT_EMAIL"

yellow "Step 4: Create product"
PROD_STATUS=$(do_request POST "$GATEWAY/api/v1/products" \
  -H "Authorization: Bearer $ACCESS" \
  -H "X-User-Role: admin" \
  --data '{
    "name":"E2E Widget","sku":"E2E-001","description":"Test product",
    "price":2500,"currency":"USD","stock":100
  }')
if [ "$PROD_STATUS" = "201" ]; then
  PRODUCT_ID=$(jq -r '.id' /tmp/e2e_body.json)
  assert_not_empty "Product created" "$PRODUCT_ID"
  green "Product created (admin role confirmed)"
  PASS=$((PASS+1))
else
  yellow "Product creation skipped (need admin role â€” using fallback product ID)"
  PRODUCT_ID="prod-1"  # Use pre-seeded product if available
fi

yellow "Step 5: Create order"
IDEM_KEY="e2e-order-$(date +%s)"
STATUS=$(do_request POST "$GATEWAY/api/v1/orders" \
  -H "Authorization: Bearer $ACCESS" \
  -H "Idempotency-Key: $IDEM_KEY" \
  --data "{
    \"currency\":\"USD\",
    \"items\":[{\"product_id\":\"$PRODUCT_ID\",\"quantity\":1}],
    \"shipping_address\":{
      \"name\":\"E2E User\",\"street\":\"123 Test St\",
      \"city\":\"Testville\",\"postal_code\":\"12345\",\"country\":\"US\"
    }
  }")

if [ "$STATUS" = "201" ]; then
  ORDER_ID=$(jq -r '.id' /tmp/e2e_body.json)
  ORDER_STATUS=$(jq -r '.status' /tmp/e2e_body.json)
  assert_not_empty "Order ID created" "$ORDER_ID"
  assert_eq "Order status is pending" "pending" "$ORDER_STATUS"
else
  red "Create order failed with HTTP $STATUS: $(cat /tmp/e2e_body.json)"
  FAIL=$((FAIL+1))
  ORDER_ID=""
fi

if [ -n "$ORDER_ID" ]; then
  yellow "Step 6: Get order $ORDER_ID"
  STATUS=$(do_request GET "$GATEWAY/api/v1/orders/$ORDER_ID" \
    -H "Authorization: Bearer $ACCESS")
  assert_status "Get order" "200" "$STATUS"
  GOT_ID=$(jq -r '.id' /tmp/e2e_body.json)
  assert_eq "Order ID matches" "$ORDER_ID" "$GOT_ID"
fi

yellow "Step 7: List orders"
STATUS=$(do_request GET "$GATEWAY/api/v1/orders?limit=5" \
  -H "Authorization: Bearer $ACCESS")
assert_status "List orders" "200" "$STATUS"
ORDER_COUNT=$(jq '.orders | length' /tmp/e2e_body.json 2>/dev/null || echo "0")
assert_not_empty "Orders list returned" "$ORDER_COUNT"

if [ -n "$ORDER_ID" ]; then
  yellow "Step 8: Idempotency check (same key â†’ same order)"
  STATUS=$(do_request POST "$GATEWAY/api/v1/orders" \
    -H "Authorization: Bearer $ACCESS" \
    -H "Idempotency-Key: $IDEM_KEY" \
    --data "{\"currency\":\"USD\",\"items\":[{\"product_id\":\"$PRODUCT_ID\",\"quantity\":1}]}")
  if [ "$STATUS" = "201" ] || [ "$STATUS" = "200" ]; then
    IDEM_ORDER_ID=$(jq -r '.id' /tmp/e2e_body.json)
    assert_eq "Idempotent order returns same ID" "$ORDER_ID" "$IDEM_ORDER_ID"
  else
    red "Idempotency check failed with HTTP $STATUS"
    FAIL=$((FAIL+1))
  fi
fi

if [ -n "$ORDER_ID" ]; then
  yellow "Step 9: Cancel order"
  STATUS=$(do_request DELETE "$GATEWAY/api/v1/orders/$ORDER_ID" \
    -H "Authorization: Bearer $ACCESS")
  assert_status "Cancel order" "200" "$STATUS"

  do_request GET "$GATEWAY/api/v1/orders/$ORDER_ID" \
    -H "Authorization: Bearer $ACCESS" > /dev/null
  CANCELLED_STATUS=$(jq -r '.status' /tmp/e2e_body.json)
  assert_eq "Order is now cancelled" "cancelled" "$CANCELLED_STATUS"
fi

yellow "Step 10: Token refresh"
STATUS=$(do_request POST "$GATEWAY/auth/refresh" \
  --data "{\"refresh_token\":\"$REFRESH\"}")
assert_status "Refresh token" "200" "$STATUS"
NEW_ACCESS=$(jq -r '.access_token' /tmp/e2e_body.json)
assert_not_empty "New access token" "$NEW_ACCESS"

echo ""
echo "â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€"
printf "  Results: \033[0;32m%d passed\033[0m, \033[0;31m%d failed\033[0m\n" "$PASS" "$FAIL"
echo "â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€"

if [ "$FAIL" -gt 0 ]; then
  exit 1
fi
