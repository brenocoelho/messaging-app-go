package grpc

import (
	"github.com/brenocoelho/messaging-app-go/internal/services"
	pb "github.com/brenocoelho/messaging-app-go/proto"
	"google.golang.org/grpc"
)

type GRPCServer struct {
	messagesServer *MessagesGRPCServer
	chatsServer    *ChatsGRPCServer
	usersServer    *UsersGRPCServer
}

func NewGRPCServer(
	messagesService services.MessagesService,
	chatsService services.ChatsService,
	usersService services.UsersService,
	realtimeService services.RealtimeService,
) *GRPCServer {
	return &GRPCServer{
		messagesServer: NewMessagesGRPCServer(messagesService, realtimeService),
		chatsServer:    NewChatsGRPCServer(chatsService),
		usersServer:    NewUsersGRPCServer(usersService),
	}
}

func (s *GRPCServer) RegisterServices(server *grpc.Server) {
	pb.RegisterMessagesServiceServer(server, s.messagesServer)
	pb.RegisterChatsServiceServer(server, s.chatsServer)
	pb.RegisterUsersServiceServer(server, s.usersServer)
}
