package grpc

import (
	"context"
	"io"
	"log/slog"

	"github.com/brenocoelho/messaging-app-go/internal/models"
	"github.com/brenocoelho/messaging-app-go/internal/services"
	"github.com/brenocoelho/messaging-app-go/pkg/jwt"
	pb "github.com/brenocoelho/messaging-app-go/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type MessagesGRPCServer struct {
	pb.UnimplementedMessagesServiceServer
	messagesService services.MessagesService
	realtimeService services.RealtimeService
}

func NewMessagesGRPCServer(messagesService services.MessagesService, realtimeService services.RealtimeService) *MessagesGRPCServer {
	return &MessagesGRPCServer{
		messagesService: messagesService,
		realtimeService: realtimeService,
	}
}

func (s *MessagesGRPCServer) SendMessage(ctx context.Context, req *pb.SendMessageRequest) (*pb.SendMessageResponse, error) {
	if req.ChatId == "" || req.Content == "" || req.IdempotencyKey == "" {
		return nil, status.Error(codes.InvalidArgument, "chat_id, content, and idempotency_key are required")
	}

	userID, _, _, err := jwt.GetUserFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "failed to get user from context: %v", err)
	}

	resp, err := s.messagesService.SendMessage(ctx, models.SendMessageRequest{
		UserID:         userID,
		ChatID:         req.ChatId,
		Content:        req.Content,
		IdempotencyKey: req.IdempotencyKey,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to send message: %v", err)
	}

	message := &pb.Message{
		Id:      resp.MessageID,
		ChatId:  req.ChatId,
		UserId:  userID,
		Content: req.Content,
		SentAt:  timestamppb.Now(),
		Status:  "SENT",
	}

	return &pb.SendMessageResponse{
		Message: message,
	}, nil
}

func (s *MessagesGRPCServer) ListMessages(ctx context.Context, req *pb.ListMessagesRequest) (*pb.ListMessagesResponse, error) {
	if req.ChatId == "" {
		return nil, status.Error(codes.InvalidArgument, "chat_id is required")
	}

	userID, _, _, err := jwt.GetUserFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "failed to get user from context: %v", err)
	}

	resp, err := s.messagesService.ListMessages(ctx, models.ListMessagesRequest{
		UserID: userID,
		ChatID: req.ChatId,
		Pagination: models.Pagination{
			Page:  req.Page,
			Limit: req.Limit,
		},
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list messages: %v", err)
	}

	var messages []*pb.Message
	for _, msg := range resp.Messages {
		message := &pb.Message{
			Id:      msg.ID,
			ChatId:  msg.ChatID,
			UserId:  msg.UserID,
			Content: msg.Body,
			SentAt:  timestamppb.New(msg.CreatedAt),
			Status:  string(msg.Status),
		}
		messages = append(messages, message)
	}

	return &pb.ListMessagesResponse{
		Messages: messages,
		Total:    resp.Total,
	}, nil
}

func (s *MessagesGRPCServer) UpdateMessageStatus(ctx context.Context, req *pb.UpdateMessageStatusRequest) (*pb.UpdateMessageStatusResponse, error) {
	if req.MessageId == "" || req.Status == "" {
		return nil, status.Error(codes.InvalidArgument, "message_id and status are required")
	}

	userID, _, _, err := jwt.GetUserFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "failed to get user from context: %v", err)
	}

	resp, err := s.messagesService.UpdateMessageStatus(ctx, models.UpdateMessageStatusRequest{
		UserID:    userID,
		MessageID: req.MessageId,
		Status:    models.MessageStatus(req.Status),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update message status: %v", err)
	}

	return &pb.UpdateMessageStatusResponse{
		MessageId: resp.MessageID,
		Status:    string(resp.Status),
	}, nil
}

func (s *MessagesGRPCServer) SubscribeToChat(req *pb.SubscribeToChatRequest, stream pb.MessagesService_SubscribeToChatServer) error {
	ctx := stream.Context()

	if req.ChatId == "" {
		return status.Error(codes.InvalidArgument, "chat_id is required")
	}

	userID, username, _, err := jwt.GetUserFromContext(ctx)
	if err != nil {
		return status.Errorf(codes.Unauthenticated, "failed to get user from context: %v", err)
	}

	slog.Info("User subscribing to chat", "userID", userID, "username", username, "chatID", req.ChatId)

	msgChan, err := s.realtimeService.SubscribeToChat(ctx, req.ChatId, userID)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to subscribe to chat: %v", err)
	}

	connectionMsg := &pb.ChatMessage{
		MessageId: "connection",
		ChatId:    req.ChatId,
		UserId:    userID,
		Username:  username,
		Content:   "Connected to chat",
		SentAt:    timestamppb.Now(),
		Status:    "CONNECTED",
	}

	if err := stream.Send(connectionMsg); err != nil {
		slog.Error("Failed to send connection message", "error", err, "userID", userID, "chatID", req.ChatId)
		return status.Errorf(codes.Internal, "failed to send connection message: %v", err)
	}

	for {
		select {
		case msg := <-msgChan:
			if msg == nil {
				slog.Info("User unsubscribed from chat", "userID", userID, "chatID", req.ChatId)
				return nil
			}

			pbMsg := &pb.ChatMessage{
				MessageId: msg.MessageID,
				ChatId:    msg.ChatID,
				UserId:    msg.SenderID,
				Username:  msg.SenderUsername,
				Content:   msg.Content,
				SentAt:    timestamppb.New(msg.SentAt),
				Status:    msg.Status,
			}

			if err := stream.Send(pbMsg); err != nil {
				if err == io.EOF {
					slog.Info("Client disconnected", "userID", userID, "chatID", req.ChatId)
					return nil
				}
				slog.Error("Failed to send message to client", "error", err, "userID", userID, "chatID", req.ChatId)
				return status.Errorf(codes.Internal, "failed to send message: %v", err)
			}

		case <-ctx.Done():
			slog.Info("Context cancelled, user unsubscribing", "userID", userID, "chatID", req.ChatId)
			return nil
		}
	}
}
