# 🧪 **How to Test the Real-Time Messaging App**

This guide shows you how to test the real-time messaging application manually and using automated scripts.

## 📋 **Prerequisites**

1. **Start services**: `task docker-up`
2. **Run server**: `task run`
3. **Install grpcurl**: `go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest`

## 🚀 **Quick Testing with Scripts**

### **Automated Testing Scripts**

| Script | Purpose | Usage |
|--------|---------|-------|
| `test_realtime_working.sh` | Test real-time messaging | `./scripts/test_realtime_working.sh` |
| `test_two_clients.sh` | Simulate two users chatting | `./scripts/test_two_clients.sh` |

### **Run All Scripts**
```bash
# Make scripts executable
chmod +x scripts/*.sh

# Test real-time messaging (recommended first)
./scripts/test_realtime_working.sh

# Test two clients interaction
./scripts/test_two_clients.sh

# Test e2e workflow
./scripts/test_e2e_workflow.sh
```

## 🧪 **Manual Testing with grpcurl**

### **1. Create User & Get JWT Token**
```bash
# Create user
grpcurl -proto proto/messaging.proto -plaintext -d '{
  "username": "testuser",
  "email": "testuser@example.com",
  "password": "testpass123"
}' localhost:50051 messaging.UsersService/CreateUser

# Login to get JWT token
grpcurl -proto proto/messaging.proto -plaintext -d '{
  "email": "testuser@example.com",
  "password": "testpass123"
}' localhost:50051 messaging.UsersService/Login
```

### **2. Create Chat**
```bash
# Replace YOUR_JWT_TOKEN with the token from step 1
grpcurl -proto proto/messaging.proto -plaintext -H "authorization: Bearer YOUR_JWT_TOKEN" -d '{
  "name": "Test Chat",
  "type": "direct",
  "user_ids": ["testuser"]
}' localhost:50051 messaging.ChatsService/CreateChat
```

### **3. Test Real-Time Messaging**
```bash
# Terminal 1: Subscribe to chat (replace CHAT_ID and JWT_TOKEN)
grpcurl -proto proto/messaging.proto -plaintext \
  -H "authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{"chat_id": "CHAT_ID"}' \
  localhost:50051 messaging.MessagesService/SubscribeToChat

# Terminal 2: Send message
grpcurl -proto proto/messaging.proto -plaintext \
  -H "authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{"chat_id": "CHAT_ID", "content": "Hello real-time!", "idempotency_key": "test_123"}' \
  localhost:50051 messaging.MessagesService/SendMessage
```

## 🧪 **Unit Testing**

```bash
# Run all tests
task test

# Run with coverage
task test-coverage
```

## 🔍 **What to Look For**

### **✅ Success Indicators**
- Connection confirmation message received
- Messages arrive instantly without polling
- JWT authentication working
- Redis idempotency preventing duplicates

### **❌ Common Issues**
- **Server not running**: Run `task run`
- **Authentication failed**: Check JWT token
- **Chat not found**: Create chat first
- **User not found**: Create user first

## 📖 **Next Steps**

- See [HOWTORUN.md](./HOWTORUN.md) for running instructions
- Check the main README.md for architecture overview
- Review scripts folder for more testing scenarios 