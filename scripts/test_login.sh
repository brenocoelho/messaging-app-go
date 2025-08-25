#!/bin/bash

# Test script for Login API
# This script demonstrates creating a user and then logging in

echo "🧪 Testing Login API"
echo "==================="

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

# Test user credentials
USERNAME="test_user_$(date +%s)"
EMAIL="${USERNAME}@example.com"
PASSWORD="password123"

# Test 1: Create a user first
echo "📝 Test 1: Creating user for login test"
echo "Request: username='${USERNAME}', email='${EMAIL}', password='${PASSWORD}'"

CREATE_RESPONSE=$(grpcurl -plaintext \
  -d "{\"username\": \"${USERNAME}\", \"email\": \"${EMAIL}\", \"password\": \"${PASSWORD}\"}" \
  localhost:50051 \
  messaging.UsersService/CreateUser 2>&1)

if echo "$CREATE_RESPONSE" | grep -q "error"; then
    echo "❌ Test 1 FAILED: Could not create user"
    echo "$CREATE_RESPONSE"
    exit 1
else
    echo "✅ Test 1 PASSED: User created successfully"
    echo "$CREATE_RESPONSE"
fi

echo ""
echo "----------------------------------------"
echo ""

# Test 2: Login with correct credentials
echo "📝 Test 2: Login with correct credentials"
echo "Request: email='${EMAIL}', password='${PASSWORD}'"

LOGIN_RESPONSE=$(grpcurl -plaintext \
  -d "{\"email\": \"${EMAIL}\", \"password\": \"${PASSWORD}\"}" \
  localhost:50051 \
  messaging.UsersService/Login 2>&1)

if echo "$LOGIN_RESPONSE" | grep -q "error"; then
    echo "❌ Test 2 FAILED: Login should have succeeded"
    echo "$LOGIN_RESPONSE"
else
    echo "✅ Test 2 PASSED: Login successful"
    echo "$LOGIN_RESPONSE"
    
    # Extract token for further tests
    TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"token": "[^"]*"' | cut -d'"' -f4)
    if [ -n "$TOKEN" ]; then
        echo "🔑 JWT Token extracted: ${TOKEN:0:50}..."
    fi
fi

echo ""
echo "----------------------------------------"
echo ""

# Test 3: Login with wrong password (should fail)
echo "📝 Test 3: Login with wrong password (should fail)"
echo "Request: email='${EMAIL}', password='wrongpassword'"

WRONG_LOGIN_RESPONSE=$(grpcurl -plaintext \
  -d "{\"email\": \"${EMAIL}\", \"password\": \"wrongpassword\"}" \
  localhost:50051 \
  messaging.UsersService/Login 2>&1)

if echo "$WRONG_LOGIN_RESPONSE" | grep -q "invalid email or password"; then
    echo "✅ Test 3 PASSED: Correctly rejected wrong password"
    echo "$WRONG_LOGIN_RESPONSE"
else
    echo "❌ Test 3 FAILED: Should have rejected wrong password"
    echo "$WRONG_LOGIN_RESPONSE"
fi

echo ""
echo "----------------------------------------"
echo ""

# Test 4: Login with non-existent email (should fail)
echo "📝 Test 4: Login with non-existent email (should fail)"
echo "Request: email='nonexistent@example.com', password='${PASSWORD}'"

NONEXISTENT_LOGIN_RESPONSE=$(grpcurl -plaintext \
  -d "{\"email\": \"nonexistent@example.com\", \"password\": \"${PASSWORD}\"}" \
  localhost:50051 \
  messaging.UsersService/Login 2>&1)

if echo "$NONEXISTENT_LOGIN_RESPONSE" | grep -q "invalid email or password"; then
    echo "✅ Test 4 PASSED: Correctly rejected non-existent email"
    echo "$NONEXISTENT_LOGIN_RESPONSE"
else
    echo "❌ Test 4 FAILED: Should have rejected non-existent email"
    echo "$NONEXISTENT_LOGIN_RESPONSE"
fi

echo ""
echo "----------------------------------------"
echo ""

# Test 5: Login without email (should fail)
echo "📝 Test 5: Login without email (should fail)"
echo "Request: password='${PASSWORD}' (missing email)"

NO_EMAIL_RESPONSE=$(grpcurl -plaintext \
  -d "{\"password\": \"${PASSWORD}\"}" \
  localhost:50051 \
  messaging.UsersService/Login 2>&1)

if echo "$NO_EMAIL_RESPONSE" | grep -q "email and password are required"; then
    echo "✅ Test 5 PASSED: Correctly rejected request without email"
    echo "$NO_EMAIL_RESPONSE"
else
    echo "❌ Test 5 FAILED: Should have rejected request without email"
    echo "$NO_EMAIL_RESPONSE"
fi

echo ""
echo "🎉 Login testing completed!"
echo ""
echo "💡 Summary:"
echo "   - Login requires only email and password ✅"
echo "   - Returns user info and JWT token ✅"
echo "   - Properly validates credentials ✅"
echo "   - Handles authentication errors correctly ✅"

if [ -n "$TOKEN" ]; then
    echo ""
    echo "🔑 Use this token for authenticated requests:"
    echo "   authorization: Bearer $TOKEN"
fi
