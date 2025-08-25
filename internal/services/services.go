package services

import (
	"github.com/brenocoelho/messaging-app-go/internal/repositories"
	"github.com/brenocoelho/messaging-app-go/pkg/jwt"
	"github.com/redis/go-redis/v9"
)

type Services struct {
	Users    UsersService
	Messages MessagesService
	Chats    ChatsService
	JWT      jwt.Service
	Realtime RealtimeService
}

func NewServices(repos *repositories.Repositories, cacheClient *redis.Client, ttlMinutes int) *Services {
	jwtService := jwt.NewService()

	realtimeService := NewRealtimeService(nil)

	usersService := NewUsersService(repos.Users, jwtService)
	messagesService := NewMessagesService(repos.Messages, cacheClient, ttlMinutes, realtimeService)
	chatsService := NewChatsService(repos.Chats, repos.Users)

	return &Services{
		Users:    usersService,
		Messages: messagesService,
		Chats:    chatsService,
		JWT:      jwtService,
		Realtime: realtimeService,
	}
}
