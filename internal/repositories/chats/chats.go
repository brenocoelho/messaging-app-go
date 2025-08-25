package chats

import (
	"context"
	"crypto/rand"
	"log/slog"
	"time"

	"github.com/brenocoelho/messaging-app-go/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oklog/ulid/v2"
	"golang.org/x/sync/errgroup"
)

type ChatsRepository interface {
	Create(ctx context.Context, req models.Chat) (string, error)
	List(ctx context.Context, req models.ListChatsRequest) (models.ListChatsResponse, error)
	Get(ctx context.Context, req models.GetChatRequest) (models.Chat, error)
	AddUserToChat(ctx context.Context, userID, chatID string) error
	RemoveUserFromChat(ctx context.Context, userID, chatID string) error
	GetChatUsers(ctx context.Context, chatID string) ([]models.User, error)
}

type chatsRepository struct {
	reader  *pgxpool.Pool
	writer  *pgxpool.Pool
	entropy *ulid.MonotonicEntropy
}

func NewChatsRepository(reader, writer *pgxpool.Pool) ChatsRepository {
	entropy := ulid.Monotonic(rand.Reader, 0)
	return &chatsRepository{
		reader:  reader,
		writer:  writer,
		entropy: entropy,
	}
}

func (r *chatsRepository) Create(ctx context.Context, req models.Chat) (string, error) {
	slog.Info("Create chat", "name", req.Name)
	id := ulid.MustNew(ulid.Timestamp(time.Now()), r.entropy)

	query := "INSERT INTO chats (id, name) VALUES (@id, @name)"
	args := pgx.NamedArgs{
		"id":   id.String(),
		"name": req.Name,
	}
	_, err := r.writer.Exec(ctx, query, args)
	if err != nil {
		slog.Error("Error creating chat", "error", err)
		return id.String(), err
	}

	return id.String(), nil
}

func (r *chatsRepository) List(ctx context.Context, req models.ListChatsRequest) (models.ListChatsResponse, error) {
	var (
		chats   []models.ChatWithLastMessage
		total   int32
		page    = req.Page
		limit   = req.Limit
		orderBy = "created_at DESC"
	)

	if page < 1 {
		page = 1
	}

	if limit < 1 {
		limit = 20
	}

	slog.Info("Listing chats", "page", page, "limit", limit, "orderBy", orderBy, "userID", req.UserID)

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		query := `SELECT c.id, c.name, c.created_at, c.updated_at,
					m.id as last_message_id, m.content as last_content, 
					m.created_at as last_message_created_at, m.status as last_message_status,
					u.username as last_message_username,
					(SELECT COUNT(*) FROM messages m2 
					 WHERE m2.chat_id = c.id AND m2.status != 'READ' AND m2.user_id != @userID) as unread_count,
					(SELECT COUNT(DISTINCT uc2.user_id) FROM users_chats uc2 WHERE uc2.chat_id = c.id) as participant_count
				  FROM chats c
				  JOIN users_chats uc ON c.id = uc.chat_id
				  LEFT JOIN messages m ON c.id = m.chat_id AND m.created_at = (
					  SELECT MAX(m3.created_at) FROM messages m3 WHERE m3.chat_id = c.id
				  )
				  LEFT JOIN users u ON m.user_id = u.id
				  WHERE uc.user_id = @userID
				  ORDER BY c.` + orderBy + `
				  LIMIT @limit OFFSET @offset`
		args := pgx.NamedArgs{
			"userID": req.UserID,
			"limit":  limit,
			"offset": (page - 1) * limit,
		}
		rows, err := r.reader.Query(ctx, query, args)
		if err != nil {
			slog.Error("Error listing chats", "error", err)
			return err
		}
		defer rows.Close()

		result := []models.ChatWithLastMessage{}
		for rows.Next() {
			var chat models.ChatWithLastMessage
			var lastMessage models.Message
			var lastMessageID, lastContent, lastMessageUsername *string
			var lastMessageCreatedAt *time.Time
			var lastMessageStatus *string

			if err := rows.Scan(
				&chat.ID, &chat.Name, &chat.CreatedAt, &chat.UpdatedAt,
				&lastMessageID, &lastContent, &lastMessageCreatedAt, &lastMessageStatus,
				&lastMessageUsername, &chat.UnreadCount, &chat.ParticipantCount,
			); err != nil {
				slog.Error("Error scanning chat", "error", err)
				return err
			}

			if lastMessageID != nil {
				lastMessage.ID = *lastMessageID
				lastMessage.Body = *lastContent
				lastMessage.CreatedAt = *lastMessageCreatedAt
				lastMessage.Status = models.MessageStatus(*lastMessageStatus)
				if lastMessageUsername != nil {
					lastMessage.User = &models.User{Username: *lastMessageUsername}
				}
				chat.LastMessage = &lastMessage
			}

			result = append(result, chat)
		}
		if err := rows.Err(); err != nil {
			slog.Error("Error iterating chats", "error", err)
			return err
		}
		chats = result
		return nil
	})

	g.Go(func() error {
		query := `SELECT COUNT(DISTINCT c.id)
				  FROM chats c
				  JOIN users_chats uc ON c.id = uc.chat_id
				  WHERE uc.user_id = @userID`
		args := pgx.NamedArgs{
			"userID": req.UserID,
		}
		if err := r.reader.QueryRow(ctx, query, args).Scan(&total); err != nil {
			slog.Error("Error counting chats", "error", err)
			return err
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return models.ListChatsResponse{}, err
	}

	return models.ListChatsResponse{
		Chats: chats,
		Total: total,
	}, nil
}

func (r *chatsRepository) Get(ctx context.Context, req models.GetChatRequest) (models.Chat, error) {
	slog.Info("Get chat", "id", req.ID, "userID", req.UserID)

	query := `SELECT c.id, c.name, c.created_at, c.updated_at
			  FROM chats c
			  JOIN users_chats uc ON c.id = uc.chat_id
			  WHERE c.id = @id AND uc.user_id = @userID`
	args := pgx.NamedArgs{
		"id":     req.ID,
		"userID": req.UserID,
	}

	var chat models.Chat
	if err := r.reader.QueryRow(ctx, query, args).Scan(
		&chat.ID, &chat.Name, &chat.CreatedAt, &chat.UpdatedAt,
	); err != nil {
		if err == pgx.ErrNoRows {
			slog.Info("Chat not found", "id", req.ID)
			return models.Chat{}, nil
		}
		slog.Error("Error getting chat", "error", err)
		return models.Chat{}, err
	}

	return chat, nil
}

func (r *chatsRepository) AddUserToChat(ctx context.Context, userID, chatID string) error {
	slog.Info("Add user to chat", "userID", userID, "chatID", chatID)

	id := ulid.MustNew(ulid.Timestamp(time.Now()), r.entropy)

	query := "INSERT INTO users_chats (id, user_id, chat_id) VALUES (@id, @user_id, @chat_id)"
	args := pgx.NamedArgs{
		"id":      id.String(),
		"user_id": userID,
		"chat_id": chatID,
	}

	_, err := r.writer.Exec(ctx, query, args)
	if err != nil {
		slog.Error("Error adding user to chat", "error", err)
		return err
	}

	return nil
}

func (r *chatsRepository) RemoveUserFromChat(ctx context.Context, userID, chatID string) error {
	slog.Info("Remove user from chat", "userID", userID, "chatID", chatID)

	query := "DELETE FROM users_chats WHERE user_id = @user_id AND chat_id = @chat_id"
	args := pgx.NamedArgs{
		"user_id": userID,
		"chat_id": chatID,
	}

	_, err := r.writer.Exec(ctx, query, args)
	if err != nil {
		slog.Error("Error removing user from chat", "error", err)
		return err
	}

	return nil
}

func (r *chatsRepository) GetChatUsers(ctx context.Context, chatID string) ([]models.User, error) {
	slog.Info("Get chat users", "chatID", chatID)

	query := `SELECT u.id, u.username, u.email, u.created_at, u.updated_at
			  FROM users u
			  JOIN users_chats uc ON u.id = uc.user_id
			  WHERE uc.chat_id = @chat_id
			  ORDER BY u.username`
	args := pgx.NamedArgs{
		"chat_id": chatID,
	}

	rows, err := r.reader.Query(ctx, query, args)
	if err != nil {
		slog.Error("Error getting chat users", "error", err)
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		if err := rows.Scan(
			&user.ID, &user.Username, &user.Email, &user.CreatedAt, &user.UpdatedAt,
		); err != nil {
			slog.Error("Error scanning user", "error", err)
			return nil, err
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		slog.Error("Error iterating users", "error", err)
		return nil, err
	}

	return users, nil
}
