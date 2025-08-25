package repositories

import (
	"github.com/brenocoelho/messaging-app-go/internal/repositories/chats"
	"github.com/brenocoelho/messaging-app-go/internal/repositories/messages"
	"github.com/brenocoelho/messaging-app-go/internal/repositories/users"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repositories struct {
	Users    users.UsersRepository
	Messages messages.MessagesRepository
	Chats    chats.ChatsRepository
}

func NewRepositories(reader, writer *pgxpool.Pool) *Repositories {
	return &Repositories{
		Users:    users.NewUsersRepository(reader, writer),
		Messages: messages.NewMessagesRepository(reader, writer),
		Chats:    chats.NewChatsRepository(reader, writer),
	}
}
