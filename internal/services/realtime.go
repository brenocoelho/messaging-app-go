package services

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/brenocoelho/messaging-app-go/internal/models"
	"github.com/brenocoelho/messaging-app-go/pkg/redisconn"
)

type RealtimeService interface {
	SubscribeToChat(ctx context.Context, chatID, userID string) (<-chan *ChatMessage, error)
	UnsubscribeFromChat(chatID, userID string)
	BroadcastMessage(chatID string, message *ChatMessage)
	ConvertToChatMessage(msg models.Message) *ChatMessage
}

type realtimeService struct {
	mu sync.RWMutex

	// Chat subscriptions: chatID -> userID -> message channel
	chatSubscriptions map[string]map[string]chan *ChatMessage

	redis *redisconn.IdempotencyService
}

type ChatMessage struct {
	MessageID      string
	ChatID         string
	SenderID       string
	SenderUsername string
	Content        string
	SentAt         time.Time
	Status         string
	Type           MessageType
}

type MessageType int

const (
	MessageTypeUnspecified MessageType = iota
	MessageTypeNew
	MessageTypeRead
	MessageTypeTyping
	MessageTypeOnline
	MessageTypeOffline
)

type UserPresence struct {
	UserID   string
	Username string
	Status   string
	LastSeen time.Time
}

type TypingIndicator struct {
	UserID    string
	Username  string
	ChatID    string
	IsTyping  bool
	Timestamp time.Time
}

func NewRealtimeService(redis *redisconn.IdempotencyService) RealtimeService {
	return &realtimeService{
		chatSubscriptions: make(map[string]map[string]chan *ChatMessage),
		redis:             redis,
	}
}

func (s *realtimeService) SubscribeToChat(ctx context.Context, chatID, userID string) (<-chan *ChatMessage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	msgChan := make(chan *ChatMessage, 100) // Buffer for 100 messages

	if s.chatSubscriptions[chatID] == nil {
		s.chatSubscriptions[chatID] = make(map[string]chan *ChatMessage)
	}

	s.chatSubscriptions[chatID][userID] = msgChan

	slog.Info("User subscribed to chat", "userID", userID, "chatID", chatID)

	go func() {
		<-ctx.Done()
		s.UnsubscribeFromChat(chatID, userID)
	}()

	return msgChan, nil
}

func (s *realtimeService) UnsubscribeFromChat(chatID, userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if chatSubs, exists := s.chatSubscriptions[chatID]; exists {
		if msgChan, userExists := chatSubs[userID]; userExists {
			close(msgChan)
			delete(chatSubs, userID)

			// Remove chat if no more subscribers
			if len(chatSubs) == 0 {
				delete(s.chatSubscriptions, chatID)
			}

			slog.Info("User unsubscribed from chat", "userID", userID, "chatID", chatID)
		}
	}
}

func (s *realtimeService) BroadcastMessage(chatID string, message *ChatMessage) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if chatSubs, exists := s.chatSubscriptions[chatID]; exists {
		slog.Info("Broadcasting message to chat", "chatID", chatID, "subscribers", len(chatSubs))

		for userID, msgChan := range chatSubs {
			select {
			case msgChan <- message:
				slog.Debug("Message sent to user", "userID", userID, "chatID", chatID)
			default:
				slog.Warn("User's message channel is full, skipping", "userID", userID, "chatID", chatID)
				go s.UnsubscribeFromChat(chatID, userID)
			}
		}
	} else {
		slog.Debug("No subscribers for chat", "chatID", chatID)
	}
}

func (s *realtimeService) ConvertToChatMessage(msg models.Message) *ChatMessage {
	username := ""
	if msg.User != nil {
		username = msg.User.Username
	}

	return &ChatMessage{
		MessageID:      msg.ID,
		ChatID:         msg.ChatID,
		SenderID:       msg.UserID,
		SenderUsername: username,
		Content:        msg.Body,
		SentAt:         msg.CreatedAt,
		Status:         string(msg.Status),
		Type:           MessageTypeNew,
	}
}

func (s *realtimeService) GetSubscriberCount(chatID string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if chatSubs, exists := s.chatSubscriptions[chatID]; exists {
		return len(chatSubs)
	}
	return 0
}

func (s *realtimeService) GetTotalSubscriptions() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	total := 0
	for _, chatSubs := range s.chatSubscriptions {
		total += len(chatSubs)
	}
	return total
}
