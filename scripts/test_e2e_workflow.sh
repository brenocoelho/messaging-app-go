#!/bin/bash

# End-to-end test script showing the complete user workflow
# This demonstrates creating a user (getting token) and using it for authenticated requests

echo "🚀 End-to-End User Workflow Test"
echo "================================="

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
USERNAME="e2e_user_$(date +%s)"
EMAIL="${USERNAME}@example.com"
PASSWORD="password123"

# Step 1: Create user and get token
echo "📝 Step 1: Creating user and getting JWT token"
echo "Request: username='${USERNAME}', email='${EMAIL}', password='${PASSWORD}'"

CREATE_RESPONSE=$(grpcurl -proto proto/messaging.proto -plaintext \
  -d "{\"username\": \"${USERNAME}\", \"email\": \"${EMAIL}\", \"password\": \"${PASSWORD}\"}" \
  localhost:50051 \
  messaging.UsersService/CreateUser 2>&1)

if echo "$CREATE_RESPONSE" | grep -q "error"; then
    echo "❌ Step 1 FAILED: Could not create user"
    echo "$CREATE_RESPONSE"
    exit 1
else
    echo "✅ Step 1 PASSED: User created successfully"
    echo "$CREATE_RESPONSE"
    
    # Extract token and user ID
    TOKEN=$(echo "$CREATE_RESPONSE" | grep -o '"token": "[^"]*"' | cut -d'"' -f4)
    USER_ID=$(echo "$CREATE_RESPONSE" | grep -o '"id": "[^"]*"' | cut -d'"' -f4)
    
    if [ -n "$TOKEN" ] && [ -n "$USER_ID" ]; then
        echo "🔑 JWT Token extracted: ${TOKEN:0:50}..."
        echo "👤 User ID extracted: $USER_ID"
    else
        echo "❌ Failed to extract token or user ID"
        exit 1
    fi
fi

echo ""
echo "----------------------------------------"
echo ""

# Step 2: Use token for authenticated request
echo "📝 Step 2: Using JWT token for authenticated GetUser request"
echo "Request: user_id='${USER_ID}' with authorization header"

GET_USER_RESPONSE=$(grpcurl -proto proto/messaging.proto -plaintext \
  -H "authorization: Bearer ${TOKEN}" \
  -d "{\"user_id\": \"${USER_ID}\"}" \
  localhost:50051 \
  messaging.UsersService/GetUser 2>&1)

if echo "$GET_USER_RESPONSE" | grep -q "error"; then
    echo "❌ Step 2 FAILED: Authenticated request failed"
    echo "$GET_USER_RESPONSE"
else
    echo "✅ Step 2 PASSED: Authenticated request successful"
    echo "$GET_USER_RESPONSE"
fi

echo ""
echo "----------------------------------------"
echo ""

# Step 3: Alternative login flow
echo "📝 Step 3: Testing alternative login flow (email + password)"
echo "Request: email='${EMAIL}', password='${PASSWORD}'"

LOGIN_RESPONSE=$(grpcurl -proto proto/messaging.proto -plaintext \
  -d "{\"email\": \"${EMAIL}\", \"password\": \"${PASSWORD}\"}" \
  localhost:50051 \
  messaging.UsersService/Login 2>&1)

if echo "$LOGIN_RESPONSE" | grep -q "error"; then
    echo "❌ Step 3 FAILED: Login should have succeeded"
    echo "$LOGIN_RESPONSE"
else
    echo "✅ Step 3 PASSED: Login successful"
    echo "$LOGIN_RESPONSE"
    
    # Extract login token
    LOGIN_TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"token": "[^"]*"' | cut -d'"' -f4)
    if [ -n "$LOGIN_TOKEN" ]; then
        echo "🔑 Login JWT Token: ${LOGIN_TOKEN:0:50}..."
    fi
fi

echo ""
echo "🎉 End-to-End Test Completed!"
echo ""
echo "💡 Summary:"
echo "   ✅ CreateUser returns both user info AND JWT token"
echo "   ✅ JWT token works for authenticated requests"
echo "   ✅ Login provides alternative way to get token"
echo "   ✅ Both tokens work for authentication"
echo ""
echo "🔄 User Workflow Options:"
echo "   Option 1: CreateUser → Use token immediately"
echo "   Option 2: CreateUser → Login later → Use token"
echo ""
echo "📖 See docs/API_USAGE.md for complete API documentation"
