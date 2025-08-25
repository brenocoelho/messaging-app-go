package models

import "time"

type Chat struct {
	ID        string    `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type ChatWithLastMessage struct {
	Chat
	LastMessage      *Message `json:"last_message,omitempty"`
	UnreadCount      int      `json:"unread_count" db:"unread_count"`
	ParticipantCount int      `json:"participant_count" db:"participant_count"`
}

type UserChat struct {
	ID        string    `json:"id" db:"id"`
	UserID    string    `json:"user_id" db:"user_id"`
	ChatID    string    `json:"chat_id" db:"chat_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type CreateChatRequest struct {
	UserID string `json:"-"`
	Name   string `json:"name"`
	Email  string `json:"email"`
}

type CreateChatResponse struct {
	ChatId string `json:"chat_id"`
}

type GetChatRequest struct {
	ID     string `json:"id" validate:"required"`
	UserID string `json:"-"`
}

type ListChatsRequest struct {
	Pagination
	UserID string `json:"-"`
}

type ListChatsResponse struct {
	Chats []ChatWithLastMessage `json:"chats"`
	Total int32                 `json:"total"`
}
