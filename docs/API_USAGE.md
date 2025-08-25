# API Usage Examples

## User Management

### Create User

Creates a new user account with username, email, and password. Also returns a JWT token for immediate authentication.

**Request:**
```protobuf
CreateUserRequest {
  username: "john_doe"
  email: "john.doe@example.com"
  password: "secure_password123"
}
```

**Response:**
```protobuf
CreateUserResponse {
  user: {
    id: "01K3EZ31YQK87SXSVPPCQFZXFM"
    username: "john_doe"
    email: "john.doe@example.com"
    created_at: "2025-08-24T18:00:00Z"
  }
  token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**gRPC Client Example:**
```bash
grpcurl -plaintext \
  -d '{"username": "john_doe", "email": "john.doe@example.com", "password": "secure_password123"}' \
  localhost:50051 \
  messaging.UsersService/CreateUser
```

### Login

Authenticates a user with email and password, returns user info and JWT token.

**Request:**
```protobuf
LoginRequest {
  email: "john.doe@example.com"
  password: "secure_password123"
}
```

**Response:**
```protobuf
LoginResponse {
  user: {
    id: "01K3EZ31YQK87SXSVPPCQFZXFM"
    username: "john_doe"
    email: "john.doe@example.com"
    created_at: "2025-08-24T18:00:00Z"
  }
  token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**gRPC Client Example:**
```bash
grpcurl -plaintext \
  -d '{"email": "john.doe@example.com", "password": "secure_password123"}' \
  localhost:50051 \
  messaging.UsersService/Login
```

### Get User

Retrieves user information by ID (requires authentication).

**Request:**
```protobuf
GetUserRequest {
  user_id: "01K3EZ31YQK87SXSVPPCQFZXFM"
}
```

**Response:**
```protobuf
GetUserResponse {
  user: {
    id: "01K3EZ31YQK87SXSVPPCQFZXFM"
    username: "john_doe"
    email: "john.doe@example.com"
    created_at: "2025-08-24T18:00:00Z"
  }
}
```

**gRPC Client Example:**
```bash
grpcurl -plaintext \
  -H "authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{"user_id": "01K3EZ31YQK87SXSVPPCQFZXFM"}' \
  localhost:50051 \
  messaging.UsersService/GetUser
```

## Chat Management

### Create Chat

Creates a new chat room (requires authentication).

**Request:**
```protobuf
CreateChatRequest {
  name: "General Discussion"
  type: "group"
  user_ids: ["01K3EZ31YQK87SXSVPPCQFZXFM", "01K3EZ31YQK87SXSVPPCQFZXFN"]
}
```

**Response:**
```protobuf
CreateChatResponse {
  chat: {
    id: "01K3EZ31YQK87SXSVPPCQFZXFO"
    name: "General Discussion"
    type: "group"
    created_at: "2025-08-24T18:00:00Z"
    members: [
      {
        id: "01K3EZ31YQK87SXSVPPCQFZXFM"
        username: "john_doe"
        email: "john.doe@example.com"
      }
    ]
  }
}
```

## Messaging

### Send Message

Sends a message to a chat (requires authentication).

**Request:**
```protobuf
SendMessageRequest {
  chat_id: "01K3EZ31YQK87SXSVPPCQFZXFO"
  sender_id: "01K3EZ31YQK87SXSVPPCQFZXFM"
  content: "Hello, everyone!"
  idempotency_key: "unique_key_12345"
}
```

**Response:**
```protobuf
SendMessageResponse {
  message: {
    id: "01K3EZ31YQK87SXSVPPCQFZXFP"
    chat_id: "01K3EZ31YQK87SXSVPPCQFZXFO"
    sender_id: "01K3EZ31YQK87SXSVPPCQFZXFM"
    content: "Hello, everyone!"
    sent_at: "2025-08-24T18:00:00Z"
    status: "SENT"
  }
}
```

## Real-time Features

### Subscribe to Chat Messages

Establishes a real-time stream to receive messages from a chat.

**Request:**
```protobuf
SubscribeToChatRequest {
  chat_id: "01K3EZ31YQK87SXSVPPCQFZXFO"
  user_id: "01K3EZ31YQK87SXSVPPCQFZXFM"
}
```

**Stream Response:**
```protobuf
ChatMessage {
  message_id: "01K3EZ31YQK87SXSVPPCQFZXFP"
  chat_id: "01K3EZ31YQK87SXSVPPCQFZXFO"
  sender_id: "01K3EZ31YQK87SXSVPPCQFZXFM"
  sender_username: "john_doe"
  content: "Hello, everyone!"
  sent_at: "2025-08-24T18:00:00Z"
  status: "SENT"
  type: MESSAGE_TYPE_NEW
}
```

## Authentication

### Getting a JWT Token

You can get a JWT token in two ways:

1. **Create a user** using the `CreateUser` endpoint (returns token immediately)
2. **Login** using the `Login` endpoint with email and password

### Using JWT Token

All endpoints except `CreateUser` and `Login` require JWT authentication. Include the JWT token in the `authorization` header:

```
authorization: Bearer YOUR_JWT_TOKEN
```

**Example workflow (Option 1 - Create User):**
```bash
# 1. Create a user (gets token immediately)
grpcurl -plaintext \
  -d '{"username": "alice", "email": "alice@example.com", "password": "password123"}' \
  localhost:50051 \
  messaging.UsersService/CreateUser

# 2. Use token from creation response for authenticated requests
grpcurl -plaintext \
  -H "authorization: Bearer YOUR_JWT_TOKEN_FROM_CREATION" \
  -d '{"user_id": "01K3EZ31YQK87SXSVPPCQFZXFM"}' \
  localhost:50051 \
  messaging.UsersService/GetUser
```

**Example workflow (Option 2 - Login):**
```bash
# 1. Create a user first
grpcurl -plaintext \
  -d '{"username": "alice", "email": "alice@example.com", "password": "password123"}' \
  localhost:50051 \
  messaging.UsersService/CreateUser

# 2. Login to get token
grpcurl -plaintext \
  -d '{"email": "alice@example.com", "password": "password123"}' \
  localhost:50051 \
  messaging.UsersService/Login

# 3. Use token for authenticated requests
grpcurl -plaintext \
  -H "authorization: Bearer YOUR_JWT_TOKEN_FROM_LOGIN" \
  -d '{"user_id": "01K3EZ31YQK87SXSVPPCQFZXFM"}' \
  localhost:50051 \
  messaging.UsersService/GetUser
```

## Error Handling

All gRPC endpoints return standard gRPC status codes:

- `INVALID_ARGUMENT` - Missing or invalid request parameters
- `UNAUTHENTICATED` - Missing or invalid JWT token
- `PERMISSION_DENIED` - User doesn't have permission for the operation
- `NOT_FOUND` - Requested resource doesn't exist
- `INTERNAL` - Server-side error

Example error response:
```json
{
  "error": "rpc error: code = InvalidArgument desc = username, email, and password are required"
}
```
