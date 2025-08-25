package services

import (
	"context"
	"log/slog"

	"github.com/brenocoelho/messaging-app-go/internal/models"
	"github.com/brenocoelho/messaging-app-go/internal/repositories/chats"
	"github.com/brenocoelho/messaging-app-go/internal/repositories/users"
)

type ChatsService interface {
	CreateChat(ctx context.Context, req models.CreateChatRequest) (models.CreateChatResponse, error)
	GetChat(ctx context.Context, req models.GetChatRequest) (models.Chat, error)
	ListChats(ctx context.Context, req models.ListChatsRequest) (models.ListChatsResponse, error)
}

type chatsService struct {
	chatsRepo chats.ChatsRepository
	usersRepo users.UsersRepository
}

func NewChatsService(chatsRepo chats.ChatsRepository, usersRepo users.UsersRepository) ChatsService {
	return &chatsService{
		chatsRepo: chatsRepo,
		usersRepo: usersRepo,
	}
}

func (s *chatsService) CreateChat(ctx context.Context, req models.CreateChatRequest) (models.CreateChatResponse, error) {
	slog.Info("CreateChat service", "userID", req.UserID, "Email", req.Email)

	toUser, err := s.usersRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		slog.Error("Error getting user by email", "error", err)
		return models.CreateChatResponse{}, err
	}

	if toUser.ID == "" {
		slog.Warn("User not found", "Email", req.Email)
		return models.CreateChatResponse{}, err
	}

	chat := models.Chat{
		Name: req.Name,
	}

	chatID, err := s.chatsRepo.Create(ctx, chat)
	if err != nil {
		slog.Error("Error creating chat", "error", err)
		return models.CreateChatResponse{}, err
	}

	err = s.chatsRepo.AddUserToChat(ctx, req.UserID, chatID)
	if err != nil {
		slog.Error("Error adding user to chat", "error", err)
		return models.CreateChatResponse{}, err
	}

	err = s.chatsRepo.AddUserToChat(ctx, toUser.ID, chatID)
	if err != nil {
		slog.Error("Error adding user to chat", "error", err)
		return models.CreateChatResponse{}, err
	}

	return models.CreateChatResponse{
		ChatId: chatID,
	}, nil
}

func (s *chatsService) GetChat(ctx context.Context, req models.GetChatRequest) (models.Chat, error) {
	slog.Info("GetChat service", "chatID", req.ID, "userID", req.UserID)

	return s.chatsRepo.Get(ctx, req)
}

func (s *chatsService) ListChats(ctx context.Context, req models.ListChatsRequest) (models.ListChatsResponse, error) {
	slog.Info("ListChats service", "userID", req.UserID)

	return s.chatsRepo.List(ctx, req)
}
