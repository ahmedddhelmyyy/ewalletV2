#!/bin/bash

# Merchant Payment Gateway - Manual Test Script
# Usage: ./test-merchant.sh [command]
# Commands: create, check, web, all (default: menu)

set -e

BASE_URL="${BASE_URL:-http://localhost:8080}"
API_KEY="key_abc123"
SECRET="supersecret"
MERCHANT_ID="merch_123"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo_logo() {
  echo -e "${BLUE}"
  echo "╔═══════════════════════════════════════════════════════════╗"
  echo "║     Merchant Payment Gateway - Test Script             ║"
  echo "╚═══════════════════════════════════════════════════╝${NC}"
}

hmac_sign() {
  local body="$1"
  local timestamp="$2"
  echo -n "${body}${timestamp}" | openssl dgst -sha256 -hmac "$SECRET" -binary | xxd -p
}

wait_server() {
  echo -e "${YELLOW}Waiting for server...${NC}"
  for i in {1..10}; do
    if curl -s "$BASE_URL/api/v1/auth/login" > /dev/null 2>&1; then
      echo -e "${GREEN}Server is ready!${NC}"
      return 0
    fi
    sleep 1
  done
  echo -e "${RED}Server not responding. Start with: ./ewallet${NC}"
  exit 1
}

# Test 1: Create Transaction
test_create() {
  echo -e "\n${GREEN}[TEST 1] Create Transaction${NC}"
  echo "============================================"

  local body='{"order_id":"'"$(date +%s)"'","amount":1500,"currency":"USD","return_url":"https://shop.com/success","cancel_url":"https://shop.com/cancel"}'
  local timestamp=$(date +%s)
  local signature=$(hmac_sign "$body" "$timestamp")

  echo "Request:"
  echo "  Method: POST"
  echo "  URL: $BASE_URL/api/v1/merchant/transactions"
  echo "  Body: $body"
  echo "  Timestamp: $timestamp"
  echo "  Signature: $signature"
  echo

  local response=$(curl -s -X POST "$BASE_URL/api/v1/merchant/transactions" \
    -H "Content-Type: application/json" \
    -H "X-Api-Key: $API_KEY" \
    -H "X-Signature: $signature" \
    -H "X-Timestamp: $timestamp" \
    -H "Idempotency-Key: idem_$(date +%s)" \
    -d "$body")

  echo "Response:"
  echo "$response" | jq .

  local redirect_url=$(echo "$response" | jq -r '.data.redirect_url // empty')
  local tx_id=$(echo "$response" | jq -r '.data.transaction_id // empty')
  local status=$(echo "$response" | jq -r '.data.status // empty')

  if [ -n "$redirect_url" ]; then
    # Extract token from URL
    local token=$(echo "$redirect_url" | grep -o 'pay/[^/]*$' | cut -d'/' -f2)
    echo -e "\n${GREEN}SUCCESS${NC} - Token: $token"
    echo "Web URL: $BASE_URL/pay/$token"

    # Save for next tests
    echo "$token" > /tmp/merchant_test_token.txt
    echo "$tx_id" > /tmp/merchant_test_txid.txt
  else
    echo -e "\n${RED}FAILED${NC} - No redirect_url in response"
    echo "Response: $response"
  fi
}

# Test 2: Check Transaction
test_check() {
  echo -e "\n${GREEN}[TEST 2] Check Transaction${NC}"
  echo "============================================"

  local tx_id="${1:-$(cat /tmp/merchant_test_txid.txt 2>/dev/null)}"

  if [ -z "$tx_id" ]; then
    echo -e "${YELLOW}No transaction ID. Run 'create' first.${NC}"
    return 1
  fi

  local body=""
  local timestamp=$(date +%s)
  local signature=$(hmac_sign "$body" "$timestamp")

  echo "Request:"
  echo "  Method: GET"
  echo "  URL: $BASE_URL/api/v1/merchant/transactions/$tx_id"
  echo

  local response=$(curl -s -X GET "$BASE_URL/api/v1/merchant/transactions/$tx_id" \
    -H "X-Api-Key: $API_KEY" \
    -H "X-Signature: $signature" \
    -H "X-Timestamp: $timestamp")

  echo "Response:"
  echo "$response" | jq .
}

# Test 3: Idempotency Test
test_idempotency() {
  echo -e "\n${GREEN}[TEST 3] Idempotency Test${NC}"
  echo "============================================"

  local idem_key="idem_test_$(date +%s)"
  local body='{"order_id":"idem_test","amount":500,"currency":"USD","return_url":"https://shop.com/return"}'
  local timestamp=$(date +%s)
  local signature=$(hmac_sign "$body" "$timestamp")

  echo "First request with idempotency key: $idem_key"
  local resp1=$(curl -s -X POST "$BASE_URL/api/v1/merchant/transactions" \
    -H "Content-Type: application/json" \
    -H "X-Api-Key: $API_KEY" \
    -H "X-Signature: $signature" \
    -H "X-Timestamp: $timestamp" \
    -H "Idempotency-Key: $idem_key" \
    -d "$body")
  echo "Response: $resp1" | jq -r '.data.transaction_id // "error"'

  echo -e "\nSecond request with same idempotency key (should return same transaction)"
  local resp2=$(curl -s -X POST "$BASE_URL/api/v1/merchant/transactions" \
    -H "Content-Type: application/json" \
    -H "X-Api-Key: $API_KEY" \
    -H "X-Signature: $signature" \
    -H "X-Timestamp: $timestamp" \
    -H "Idempotency-Key: $idem_key" \
    -d "$body")
  echo "Response: $resp2" | jq -r '.data.transaction_id // "error"'

  local tx1=$(echo "$resp1" | jq -r '.data.transaction_id')
  local tx2=$(echo "$resp2" | jq -r '.data.transaction_id')

  if [ "$tx1" = "$tx2" ] && [ "$tx1" != "null" ]; then
    echo -e "\n${GREEN}SUCCESS${NC} - Idempotency working correctly"
  else
    echo -e "\n${RED}FAILED${NC} - Different transaction IDs returned"
  fi
}

# Test 4: Web View
test_web() {
  echo -e "\n${GREEN}[TEST 4] Web Payment View${NC}"
  echo "============================================"

  local token="${1:-$(cat /tmp/merchant_test_token.txt 2>/dev/null)}"

  if [ -z "$token" ]; then
    echo -e "${YELLOW}No token. Run 'create' first.${NC}"
    return 1
  fi

  echo "Opening: $BASE_URL/pay/$token"
  echo

  local response=$(curl -s -X GET "$BASE_URL/pay/$token")

  if echo "$response" | grep -q "Payment Confirmation"; then
    echo -e "${GREEN}SUCCESS${NC} - Payment page rendered"
    echo -e "Order ID: $(echo "$response" | grep -o 'Order ID[^<]*' | head -1)"
    echo -e "Amount: $(echo "$response" | grep -o '\$[^<]*' | head -1)"
  elif echo "$response" | grep -q "Expired"; then
    echo -e "${YELLOW}EXPIRED${NC} - Payment link has expired"
  else
    echo -e "${RED}FAILED${NC} - Unexpected response"
  fi
}

# Test 5: Pay Confirm (without actual payment)
test_confirm() {
  echo -e "\n${GREEN}[TEST 5] Pay Confirm (Insufficient Balance Test)${NC}"
  echo "============================================"

  local token="$(cat /tmp/merchant_test_token.txt 2>/dev/null)"

  if [ -z "$token" ]; then
    echo -e "${YELLOW}No token. Run 'create' first.${NC}"
    return 1
  fi

  echo "Testing with user_id=test_user (balance: \$10,000)"
  echo "Sending confirm with amount that exceeds balance..."

  # This would normally require form submission
  # For now, just test that the endpoint responds
  local response=$(curl -s -X POST "$BASE_URL/pay/$token/confirm" \
    -H "Content-Type: application/x-www-form-urlencoded" \
    -d "user_id=test_user&amount=99999999")

  if echo "$response" | grep -q "Insufficient"; then
    echo -e "${GREEN}SUCCESS${NC} - Insufficient balance detected"
  else
    echo -e "${YELLOW}Response preview:${NC}"
    echo "$response" | head -20
  fi
}

# Test 6: Auth Failures
test_auth_fail() {
  echo -e "\n${GREEN}[TEST 6] Authentication Failure Tests${NC}"
  echo "============================================"

  local body='{"order_id":"test","amount":100,"currency":"USD","return_url":"https://test.com"}'
  local timestamp=$(date +%s)

  echo -e "\n${YELLOW}Test 6a: Missing headers${NC}"
  curl -s -X POST "$BASE_URL/api/v1/merchant/transactions" \
    -H "Content-Type: application/json" \
    -d "$body" | jq .

  echo -e "\n${YELLOW}Test 6b: Invalid API key${NC}"
  curl -s -X POST "$BASE_URL/api/v1/merchant/transactions" \
    -H "Content-Type: application/json" \
    -H "X-Api-Key: invalid_key" \
    -H "X-Signature: abc123" \
    -H "X-Timestamp: $timestamp" \
    -d "$body" | jq .

  echo -e "\n${YELLOW}Test 6c: Invalid signature${NC}"
  curl -s -X POST "$BASE_URL/api/v1/merchant/transactions" \
    -H "Content-Type: application/json" \
    -H "X-Api-Key: $API_KEY" \
    -H "X-Signature: invalid_signature" \
    -H "X-Timestamp: $timestamp" \
    -d "$body" | jq .

  echo -e "\n${YELLOW}Test 6d: Expired timestamp${NC}"
  local old_ts=$((timestamp - 600))
  local sig=$(hmac_sign "$body" "$old_ts")
  curl -s -X POST "$BASE_URL/api/v1/merchant/transactions" \
    -H "Content-Type: application/json" \
    -H "X-Api-Key: $API_KEY" \
    -H "X-Signature: $sig" \
    -H "X-Timestamp: $old_ts" \
    -d "$body" | jq .
}

# Menu
show_menu() {
  echo_logo
  echo
  echo "Server: $BASE_URL"
  echo
  echo "Commands:"
  echo "  create     - Create a new merchant transaction"
  echo "  check      - Check transaction status (uses saved ID)"
  echo "  web        - Open payment confirmation page"
  echo "  idempotent - Test idempotency key handling"
  echo "  confirm    - Test payment confirmation flow"
  echo "  auth       - Test authentication failures"
  echo "  all        - Run all tests"
  echo "  token      - Show current saved token"
  echo
  echo "Or run with argument: ./test-merchant.sh create"
}

# Main
case "${1:-menu}" in
  menu)
    show_menu
    ;;
  create)
    wait_server
    test_create
    ;;
  check)
    wait_server
    test_check "$2"
    ;;
  web)
    wait_server
    test_web "$2"
    ;;
  idempotent|idempotency)
    wait_server
    test_idempotency
    ;;
  confirm)
    wait_server
    test_confirm
    ;;
  auth)
    wait_server
    test_auth_fail
    ;;
  all)
    wait_server
    test_create
    test_check
    test_web
    test_idempotency
    test_auth_fail
    ;;
  token)
    echo "Token: $(cat /tmp/merchant_test_token.txt 2>/dev/null || echo 'none')"
    echo "TxID:  $(cat /tmp/merchant_test_txid.txt 2>/dev/null || echo 'none')"
    ;;
  *)
    echo -e "${RED}Unknown command: $1${NC}"
    show_menu
    exit 1
    ;;
esac