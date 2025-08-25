package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"

	"github.com/brenocoelho/messaging-app-go/internal/models"
	"github.com/brenocoelho/messaging-app-go/internal/repositories/messages"
	"github.com/brenocoelho/messaging-app-go/pkg/redisconn"
	"github.com/redis/go-redis/v9"
)

type MessagesService interface {
	SendMessage(ctx context.Context, req models.SendMessageRequest) (models.SendMessageResponse, error)
	ListMessages(ctx context.Context, req models.ListMessagesRequest) (models.ListMessagesResponse, error)
	UpdateMessageStatus(ctx context.Context, req models.UpdateMessageStatusRequest) (models.UpdateMessageStatusResponse, error)
}

type messagesService struct {
	messagesRepo messages.MessagesRepository
	cache        *redis.Client
	idempotency  *redisconn.IdempotencyService
	realtime     RealtimeService
}

func NewMessagesService(messagesRepo messages.MessagesRepository, cacheClient *redis.Client, ttlMinutes int, realtime RealtimeService) MessagesService {
	return &messagesService{
		messagesRepo: messagesRepo,
		cache:        cacheClient,
		idempotency:  redisconn.NewIdempotencyService(cacheClient, ttlMinutes),
		realtime:     realtime,
	}
}

func (s *messagesService) SendMessage(ctx context.Context, req models.SendMessageRequest) (models.SendMessageResponse, error) {
	slog.Info("SendMessage service", "userID", req.UserID, "chatID", req.ChatID)

	idempotencyKey := req.IdempotencyKey
	if idempotencyKey == "" {
		hash := sha256.Sum256([]byte(req.UserID + req.ChatID + req.Content))
		idempotencyKey = hex.EncodeToString(hash[:])
	}

	exists, err := s.idempotency.CheckAndSetIdempotency(ctx, idempotencyKey)
	if err != nil {
		slog.Error("Error checking idempotency in Redis", "error", err)
		return models.SendMessageResponse{}, err
	}

	if exists {
		slog.Info("Message already exists with idempotency key", "idempotencyKey", idempotencyKey)
		return models.SendMessageResponse{}, fmt.Errorf("message with idempotency key already exists")
	}

	message := models.Message{
		IdempotencyKey: idempotencyKey,
		ChatID:         req.ChatID,
		UserID:         req.UserID,
		Body:           req.Content,
		Status:         models.MessageStatusSent,
	}

	messageID, err := s.messagesRepo.Send(ctx, message)
	if err != nil {
		slog.Error("Error sending message", "error", err)
		return models.SendMessageResponse{}, err
	}

	// Broadcast message to real-time subscribers
	if s.realtime != nil {
		msg, err := s.messagesRepo.Get(ctx, messageID)
		if err != nil {
			slog.Warn("Failed to get message for broadcasting", "error", err, "messageID", messageID)
		} else {
			chatMsg := s.realtime.ConvertToChatMessage(msg)
			s.realtime.BroadcastMessage(req.ChatID, chatMsg)
			slog.Info("Message broadcasted to real-time subscribers", "chatID", req.ChatID, "messageID", messageID)
		}
	}

	return models.SendMessageResponse{
		MessageID: messageID,
	}, nil
}

func (s *messagesService) ListMessages(ctx context.Context, req models.ListMessagesRequest) (models.ListMessagesResponse, error) {
	slog.Info("ListMessages service", "userID", req.UserID, "chatID", req.ChatID)

	return s.messagesRepo.List(ctx, req)
}

func (s *messagesService) UpdateMessageStatus(ctx context.Context, req models.UpdateMessageStatusRequest) (models.UpdateMessageStatusResponse, error) {
	slog.Info("UpdateMessageStatus service", "messageID", req.MessageID, "status", req.Status)

	var err error
	switch req.Status {
	case models.MessageStatusRead:
		err = s.messagesRepo.MarkAsRead(ctx, req.MessageID, req.UserID)
	case models.MessageStatusDelivered:
		err = s.messagesRepo.MarkAsDelivered(ctx, req.MessageID)
	default:
		return models.UpdateMessageStatusResponse{}, fmt.Errorf("invalid status: %s", req.Status)
	}

	if err != nil {
		slog.Error("Error updating message status", "error", err)
		return models.UpdateMessageStatusResponse{}, err
	}

	return models.UpdateMessageStatusResponse{
		MessageID: req.MessageID,
		Status:    req.Status,
	}, nil
}
