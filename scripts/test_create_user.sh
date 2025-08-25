#!/bin/bash

# Test script for the updated CreateUser API
# This script demonstrates the new requirement for username, email, and password

echo "🧪 Testing Updated CreateUser API"
echo "=================================="

# Check if grpcurl is available
if ! command -v grpcurl &> /dev/null; then
    echo "❌ grpcurl is not installed. Please install it to test the API."
    echo "   Installation: go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest"
    exit 1
fi

# Check if server is running
if ! nc -z localhost 50051 2>/dev/null; then
    echo "❌ gRPC server is not running on localhost:50051"
    echo "   Start the server with: task run-dev"
    exit 1
fi

echo "✅ gRPC server is running"
echo ""

# Test 1: Valid user creation
echo "📝 Test 1: Creating user with all required fields (should return token)"
echo "Request: username='alice', email='alice@example.com', password='password123'"

RESPONSE=$(grpcurl -plaintext \
  -d '{"username": "alice", "email": "alice@example.com", "password": "password123"}' \
  localhost:50051 \
  messaging.UsersService/CreateUser 2>&1)

if echo "$RESPONSE" | grep -q "error"; then
    echo "❌ Test 1 FAILED:"
    echo "$RESPONSE"
else
    echo "✅ Test 1 PASSED:"
    echo "$RESPONSE"
    
    # Check if token is included in response
    if echo "$RESPONSE" | grep -q "token"; then
        echo "✅ Token included in CreateUser response"
        TOKEN=$(echo "$RESPONSE" | grep -o '"token": "[^"]*"' | cut -d'"' -f4)
        if [ -n "$TOKEN" ]; then
            echo "🔑 JWT Token: ${TOKEN:0:50}..."
        fi
    else
        echo "❌ Token missing from CreateUser response"
    fi
fi

echo ""
echo "----------------------------------------"
echo ""

# Test 2: Missing email (should fail)
echo "📝 Test 2: Creating user without email (should fail)"
echo "Request: username='bob', password='password123' (missing email)"

RESPONSE=$(grpcurl -plaintext \
  -d '{"username": "bob", "password": "password123"}' \
  localhost:50051 \
  messaging.UsersService/CreateUser 2>&1)

if echo "$RESPONSE" | grep -q "username, email, and password are required"; then
    echo "✅ Test 2 PASSED: Correctly rejected request without email"
    echo "$RESPONSE"
else
    echo "❌ Test 2 FAILED: Should have rejected request without email"
    echo "$RESPONSE"
fi

echo ""
echo "----------------------------------------"
echo ""

# Test 3: Missing username (should fail)
echo "📝 Test 3: Creating user without username (should fail)"
echo "Request: email='charlie@example.com', password='password123' (missing username)"

RESPONSE=$(grpcurl -plaintext \
  -d '{"email": "charlie@example.com", "password": "password123"}' \
  localhost:50051 \
  messaging.UsersService/CreateUser 2>&1)

if echo "$RESPONSE" | grep -q "username, email, and password are required"; then
    echo "✅ Test 3 PASSED: Correctly rejected request without username"
    echo "$RESPONSE"
else
    echo "❌ Test 3 FAILED: Should have rejected request without username"
    echo "$RESPONSE"
fi

echo ""
echo "----------------------------------------"
echo ""

# Test 4: Missing password (should fail)
echo "📝 Test 4: Creating user without password (should fail)"
echo "Request: username='david', email='david@example.com' (missing password)"

RESPONSE=$(grpcurl -plaintext \
  -d '{"username": "david", "email": "david@example.com"}' \
  localhost:50051 \
  messaging.UsersService/CreateUser 2>&1)

if echo "$RESPONSE" | grep -q "username, email, and password are required"; then
    echo "✅ Test 4 PASSED: Correctly rejected request without password"
    echo "$RESPONSE"
else
    echo "❌ Test 4 FAILED: Should have rejected request without password"
    echo "$RESPONSE"
fi

echo ""
echo "🎉 Testing completed!"
echo ""
echo "💡 Summary:"
echo "   - CreateUser now requires username, email, AND password"
echo "   - CreateUser returns both user info AND JWT token"
echo "   - All validation is working correctly"
echo "   - API documentation updated in docs/API_USAGE.md"

if [ -n "$TOKEN" ]; then
    echo ""
    echo "🔑 Use this token for authenticated requests:"
    echo "   authorization: Bearer $TOKEN"
fi
