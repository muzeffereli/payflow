#!/usr/bin/env bash

set -euo pipefail

GATEWAY="${GATEWAY:-http://localhost:8080}"
CONCURRENCY="${CONCURRENCY:-100}"
REQUESTS_PER_USER="${REQUESTS_PER_USER:-10}"
TOTAL_REQUESTS=$((CONCURRENCY * REQUESTS_PER_USER))
PRODUCT_ID="${PRODUCT_ID:-prod-1}"

yellow() { printf '\033[1;33mÂ» %s\033[0m\n' "$*"; }
green()  { printf '\033[0;32mâœ“ %s\033[0m\n' "$*"; }
red()    { printf '\033[0;31mâœ— %s\033[0m\n' "$*"; }

if [ -z "${ACCESS_TOKEN:-}" ]; then
  yellow "No ACCESS_TOKEN set â€” registering a load test user..."
  LT_EMAIL="loadtest-$(date +%s)@test.com"
  RESP=$(curl -s -X POST "$GATEWAY/auth/register" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"$LT_EMAIL\",\"name\":\"Load Tester\",\"password\":\"LoadTest1!\"}")
  ACCESS_TOKEN=$(echo "$RESP" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)
  if [ -z "$ACCESS_TOKEN" ]; then
    red "Failed to get access token. Response: $RESP"
    exit 1
  fi
  green "Registered load test user: $LT_EMAIL"
fi

echo ""
yellow "Load Test Configuration"
echo "  Gateway:        $GATEWAY"
echo "  Concurrency:    $CONCURRENCY"
echo "  Total requests: $TOTAL_REQUESTS"
echo "  Product ID:     $PRODUCT_ID"
echo ""

if command -v k6 &>/dev/null; then
  TOOL="k6"
elif command -v hey &>/dev/null; then
  TOOL="hey"
else
  red "Neither 'k6' nor 'hey' is installed."
  echo ""
  echo "Install options:"
  echo "  hey: go install github.com/rakyll/hey@latest"
  echo "  k6:  https://k6.io/docs/getting-started/installation/"
  exit 1
fi

if [ "$TOOL" = "hey" ]; then
  yellow "Running load test with hey..."

  echo ""
  yellow "Test 1: Health check (baseline)"
  hey -n "$TOTAL_REQUESTS" -c "$CONCURRENCY" \
    "$GATEWAY/healthz"

  echo ""
  yellow "Test 2: List orders (DB read)"
  hey -n "$((TOTAL_REQUESTS / 2))" -c "$CONCURRENCY" \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    "$GATEWAY/api/v1/orders?limit=10"

  echo ""
  yellow "Test 3: Create order (write path)"
  hey -n "$((TOTAL_REQUESTS / 4))" -c "$((CONCURRENCY / 4))" \
    -m POST \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    -H "Content-Type: application/json" \
    -H "Idempotency-Key: load-test-fixed-key" \
    -d "{\"currency\":\"USD\",\"items\":[{\"product_id\":\"$PRODUCT_ID\",\"quantity\":1}]}" \
    "$GATEWAY/api/v1/orders"

elif [ "$TOOL" = "k6" ]; then
  yellow "Running load test with k6..."

  K6_SCRIPT="/tmp/load_test_k6.js"
  cat > "$K6_SCRIPT" << HEREDOC
import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Rate, Trend } from 'k6/metrics';

const errorRate = new Rate('errors');
const orderLatency = new Trend('order_latency');

export const options = {
  scenarios: {
    health_check: {
      executor: 'constant-vus',
      vus: ${CONCURRENCY},
      duration: '30s',
      exec: 'healthCheck',
    },
    list_orders: {
      executor: 'constant-vus',
      vus: ${CONCURRENCY},
      duration: '30s',
      startTime: '35s',
      exec: 'listOrders',
    },
    create_orders: {
      executor: 'constant-vus',
      vus: $((CONCURRENCY / 4)),
      duration: '60s',
      startTime: '70s',
      exec: 'createOrder',
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<500'],  // 95% of requests under 500ms
    errors: ['rate<0.01'],              // Error rate below 1%
    http_req_failed: ['rate<0.01'],
  },
};

const GATEWAY = '${GATEWAY}';
const ACCESS_TOKEN = '${ACCESS_TOKEN}';
const PRODUCT_ID = '${PRODUCT_ID}';

export function healthCheck() {
  const res = http.get(\`\${GATEWAY}/healthz\`);
  check(res, { 'status 200': (r) => r.status === 200 });
  errorRate.add(res.status !== 200);
}

export function listOrders() {
  const res = http.get(\`\${GATEWAY}/api/v1/orders?limit=10\`, {
    headers: { 'Authorization': \`Bearer \${ACCESS_TOKEN}\` },
  });
  check(res, { 'status 200': (r) => r.status === 200 });
  errorRate.add(res.status !== 200);
}

export function createOrder() {
  const key = \`load-\${__VU}-\${__ITER}\`;
  const start = Date.now();
  const res = http.post(\`\${GATEWAY}/api/v1/orders\`, JSON.stringify({
    currency: 'USD',
    items: [{ product_id: PRODUCT_ID, quantity: 1 }],
  }), {
    headers: {
      'Authorization': \`Bearer \${ACCESS_TOKEN}\`,
      'Content-Type': 'application/json',
      'Idempotency-Key': key,
    },
  });
  orderLatency.add(Date.now() - start);
  check(res, { 'order created': (r) => r.status === 201 || r.status === 200 });
  errorRate.add(res.status !== 201 && res.status !== 200);
  sleep(0.1);
}
HEREDOC

  k6 run "$K6_SCRIPT"
fi

echo ""
green "Load test complete."
