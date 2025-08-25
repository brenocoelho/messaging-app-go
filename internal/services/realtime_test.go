package services

import (
	"context"
	"testing"
	"time"

	"github.com/brenocoelho/messaging-app-go/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRealtimeService(t *testing.T) {
	service := NewRealtimeService(nil)
	assert.NotNil(t, service)

	// Test that the service implements the interface
	var _ RealtimeService = service
}

func TestRealtimeService_SubscribeToChat(t *testing.T) {
	service := NewRealtimeService(nil)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Subscribe to chat with unique ID
	msgChan, err := service.SubscribeToChat(ctx, "chat_subscribe_test", "user_subscribe_test")
	require.NoError(t, err)
	assert.NotNil(t, msgChan)

	// Ensure cleanup happens
	cancel()
	time.Sleep(10 * time.Millisecond) // Give time for cleanup goroutine
}

func TestRealtimeService_UnsubscribeFromChat(t *testing.T) {
	service := NewRealtimeService(nil)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Subscribe first with unique ID
	_, err := service.SubscribeToChat(ctx, "chat_unsub_test", "user_unsub_test")
	require.NoError(t, err)

	// Unsubscribe
	service.UnsubscribeFromChat("chat_unsub_test", "user_unsub_test")

	// Test that we can still create new subscriptions
	newChan, err := service.SubscribeToChat(ctx, "chat_unsub_test", "user_unsub_test2")
	require.NoError(t, err)
	assert.NotNil(t, newChan)

	cancel()
	time.Sleep(10 * time.Millisecond) // Give time for cleanup
}

func TestRealtimeService_BroadcastMessage(t *testing.T) {
	service := NewRealtimeService(nil)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Subscribe two users to chat with unique ID
	user1Chan, err := service.SubscribeToChat(ctx, "chat_broadcast_test", "user_broadcast_1")
	require.NoError(t, err)
	user2Chan, err := service.SubscribeToChat(ctx, "chat_broadcast_test", "user_broadcast_2")
	require.NoError(t, err)

	// Create test message
	message := &ChatMessage{
		MessageID:      "msg_broadcast_test",
		ChatID:         "chat_broadcast_test",
		SenderID:       "user_broadcast_1",
		SenderUsername: "Alice",
		Content:        "Hello everyone!",
		SentAt:         time.Now(),
		Status:         "SENT",
		Type:           MessageTypeNew,
	}

	// Broadcast message
	service.BroadcastMessage("chat_broadcast_test", message)

	// Check if both users received the message
	select {
	case receivedMsg := <-user1Chan:
		assert.Equal(t, message.MessageID, receivedMsg.MessageID)
		assert.Equal(t, message.Content, receivedMsg.Content)
	case <-time.After(100 * time.Millisecond):
		t.Error("User1 did not receive message")
	}

	select {
	case receivedMsg := <-user2Chan:
		assert.Equal(t, message.MessageID, receivedMsg.MessageID)
		assert.Equal(t, message.Content, receivedMsg.Content)
	case <-time.After(100 * time.Millisecond):
		t.Error("User2 did not receive message")
	}

	cancel()
	time.Sleep(10 * time.Millisecond) // Give time for cleanup
}

func TestRealtimeService_ConvertToChatMessage(t *testing.T) {
	service := NewRealtimeService(nil)

	// Create a test message
	msg := models.Message{
		ID:        "msg123",
		ChatID:    "chat123",
		UserID:    "user123",
		Body:      "Test message",
		Status:    models.MessageStatusSent,
		CreatedAt: time.Now(),
	}

	// Convert to chat message
	chatMsg := service.ConvertToChatMessage(msg)

	assert.Equal(t, msg.ID, chatMsg.MessageID)
	assert.Equal(t, msg.ChatID, chatMsg.ChatID)
	assert.Equal(t, msg.UserID, chatMsg.SenderID)
	assert.Equal(t, msg.Body, chatMsg.Content)
	assert.Equal(t, string(msg.Status), chatMsg.Status)
	assert.Equal(t, MessageTypeNew, chatMsg.Type)
}

func TestRealtimeService_ContextCancellation(t *testing.T) {
	service := NewRealtimeService(nil)

	// Create context with cancellation
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)

	// Subscribe to chat with unique ID
	_, err := service.SubscribeToChat(ctx, "chat_context_test", "user_context_test")
	require.NoError(t, err)

	// Cancel context
	cancel()

	// Wait a bit for cleanup
	time.Sleep(100 * time.Millisecond)

	// Test that we can create new subscriptions after cancellation
	newCtx, newCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer newCancel()

	newChan, err := service.SubscribeToChat(newCtx, "chat_context_test", "user_context_test2")
	require.NoError(t, err)
	assert.NotNil(t, newChan)

	newCancel()
	time.Sleep(10 * time.Millisecond) // Give time for cleanup
}
