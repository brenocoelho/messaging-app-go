package grpc

import (
	"context"

	"github.com/brenocoelho/messaging-app-go/internal/models"
	"github.com/brenocoelho/messaging-app-go/internal/services"
	"github.com/brenocoelho/messaging-app-go/pkg/jwt"
	pb "github.com/brenocoelho/messaging-app-go/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ChatsGRPCServer struct {
	pb.UnimplementedChatsServiceServer
	chatsService services.ChatsService
}

func NewChatsGRPCServer(chatsService services.ChatsService) *ChatsGRPCServer {
	return &ChatsGRPCServer{
		chatsService: chatsService,
	}
}

func (s *ChatsGRPCServer) CreateChat(ctx context.Context, req *pb.CreateChatRequest) (*pb.CreateChatResponse, error) {
	if req.Name == "" || req.Email == "" {
		return nil, status.Error(codes.InvalidArgument, "name and user_ids are required")
	}

	userID, _, _, err := jwt.GetUserFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "failed to get user from context: %v", err)
	}

	resp, err := s.chatsService.CreateChat(ctx, models.CreateChatRequest{
		UserID: userID,
		Name:   req.Name,
		Email:  req.Email,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create chat: %v", err)
	}

	return &pb.CreateChatResponse{
		ChatId: resp.ChatId,
	}, nil
}

func (s *ChatsGRPCServer) GetChat(ctx context.Context, req *pb.GetChatRequest) (*pb.GetChatResponse, error) {
	if req.ChatId == "" {
		return nil, status.Error(codes.InvalidArgument, "chat_id is required")
	}

	userID, _, _, err := jwt.GetUserFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "failed to get user from context: %v", err)
	}

	chat, err := s.chatsService.GetChat(ctx, models.GetChatRequest{
		ID:     req.ChatId,
		UserID: userID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get chat: %v", err)
	}

	if chat.ID == "" {
		return nil, status.Error(codes.NotFound, "chat not found")
	}

	pbChat := &pb.Chat{
		Id:          chat.ID,
		Name:        chat.Name,
		CreatedAt:   timestamppb.New(chat.CreatedAt),
		Members:     []*pb.User{},
		LastMessage: nil,
	}

	return &pb.GetChatResponse{
		Chat: pbChat,
	}, nil
}

func (s *ChatsGRPCServer) ListChats(ctx context.Context, req *pb.ListChatsRequest) (*pb.ListChatsResponse, error) {
	userID, _, _, err := jwt.GetUserFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "failed to get user from context: %v", err)
	}

	resp, err := s.chatsService.ListChats(ctx, models.ListChatsRequest{
		UserID: userID,
		Pagination: models.Pagination{
			Page:  req.Page,
			Limit: req.Limit,
		},
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list chats: %v", err)
	}

	chats := make([]*pb.Chat, len(resp.Chats))
	for i, chat := range resp.Chats {
		var lastMessage *pb.Message
		if chat.LastMessage != nil {
			lastMessage = &pb.Message{
				Id:      chat.LastMessage.ID,
				ChatId:  chat.LastMessage.ChatID,
				UserId:  chat.LastMessage.UserID,
				Content: chat.LastMessage.Body,
				SentAt:  timestamppb.New(chat.LastMessage.CreatedAt),
				Status:  string(chat.LastMessage.Status),
			}
		}

		chats[i] = &pb.Chat{
			Id:          chat.ID,
			Name:        chat.Name,
			CreatedAt:   timestamppb.New(chat.CreatedAt),
			Members:     []*pb.User{},
			LastMessage: lastMessage,
		}
	}

	return &pb.ListChatsResponse{
		Chats: chats,
		Total: resp.Total,
	}, nil
}
