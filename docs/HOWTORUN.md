# 🚀 **How to Run the Real-Time Messaging App**

This guide shows you how to run the application using the provided Taskfile commands.

## 📋 **Prerequisites**

- Go 1.21+
- Docker & Docker Compose
- Task (task runner)

## 🛠 **Install Task Runner**

```bash
# Install Task runner
go install github.com/go-task/task/v3/cmd/task@latest

# Or using package manager
# Ubuntu/Debian: sudo snap install task --classic
# macOS: brew install go-task
```

## 🚀 **Quick Start**

### **1. Start Services**
```bash
# Start PostgreSQL and Redis
task local-up
```

### **2. Run the Application**
```bash
# Build and run the gRPC server
task run
```

### **3. Test the Application**
```bash
# In another terminal, test the real-time messaging
./scripts/test_realtime_working.sh
```

## 📚 **Available Task Commands**

| Command | Description |
|---------|-------------|
| `task build` | Build the gRPC server |
| `task run` | Build and run the server |
| `task test` | Run all unit tests |
| `task local-up` | Start local services (PostgreSQL, Redis) |
| `task local-down` | Stop local services |
| `task proto` | Generate protobuf files |
| `task clean` | Clean build artifacts |
| `task dev` | Start development environment |

## 🔧 **Manual Commands**

If you prefer not to use Task:

```bash
# Start services
docker-compose -f docker/local/docker-compose.yml up -d

# Build manually
go build -o bin/grpc-server cmd/grpc-server/main.go

# Run manually
./bin/grpc-server
```

## 🌐 **Access Points**

- **gRPC Server**: `localhost:50051`
- **PostgreSQL**: `localhost:5432`
- **Redis**: `localhost:6379`

## 📖 **Next Steps**

- See [HOWTOTEST.md](./HOWTOTEST.md) for testing instructions
- Check the scripts folder for automated testing
- Review the architecture in the main README.md 