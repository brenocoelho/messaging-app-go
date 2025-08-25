package models

import "time"

type MessageStatus string

const (
	MessageStatusSent      MessageStatus = "SENT"
	MessageStatusRead      MessageStatus = "READ"
	MessageStatusDelivered MessageStatus = "DELIVERED"
)

type Message struct {
	ID             string        `json:"id" db:"id"`
	IdempotencyKey string        `json:"idempotency_key" db:"idempotency_key"`
	UserID         string        `json:"user_id" db:"user_id"`
	ChatID         string        `json:"chat_id" db:"chat_id"`
	Body           string        `json:"content" db:"content"`
	Status         MessageStatus `json:"status" db:"status"`
	CreatedAt      time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at" db:"updated_at"`
	User           *User         `json:"user,omitempty"`
}

type SendMessageRequest struct {
	UserID         string `json:"-"`
	ChatID         string `json:"chat_id" validate:"required"`
	Content        string `json:"content" validate:"required"`
	IdempotencyKey string `json:"idempotency_key" validate:"required"`
}

type SendMessageResponse struct {
	MessageID string `json:"message_id"`
}

type ReadMessageRequest struct {
	UserID    string `json:"-"`
	MessageID string `json:"message_id" validate:"required"`
}

type ReadMessageResponse struct {
	Content string `json:"content"`
	ChatID  string `json:"chat_id"`
}

type ListMessagesRequest struct {
	Pagination
	UserID string     `json:"-"`
	ChatID string     `json:"chat_id" validate:"required"`
	Before *time.Time `json:"before,omitempty"`
	After  *time.Time `json:"after,omitempty"`
}

type ListMessagesResponse struct {
	Messages []Message `json:"messages"`
	Total    int32     `json:"total"`
}

type ListChatMessagesRequest struct {
	Pagination
	UserID string `json:"-"`
	ChatID string `json:"chat_id" validate:"required"`
}

type ListChatMessagesResponse struct {
	Messages []Message `json:"messages"`
	Total    int       `json:"total"`
}

type UpdateMessageStatusRequest struct {
	UserID    string        `json:"-"`
	MessageID string        `json:"message_id" validate:"required"`
	Status    MessageStatus `json:"status" validate:"required,oneof='SENT' 'READ' 'DELIVERED'"`
}

type UpdateMessageStatusResponse struct {
	MessageID string        `json:"message_id"`
	Status    MessageStatus `json:"status"`
}
