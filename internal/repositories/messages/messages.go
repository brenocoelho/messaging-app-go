package messages

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

type MessagesRepository interface {
	Send(ctx context.Context, req models.Message) (string, error)
	List(ctx context.Context, req models.ListMessagesRequest) (models.ListMessagesResponse, error)
	Get(ctx context.Context, messageID string) (models.Message, error)
	MarkAsRead(ctx context.Context, messageID, userID string) error
	MarkAsDelivered(ctx context.Context, messageID string) error
	GetByIdempotencyKey(ctx context.Context, idempotencyKey string) (models.Message, error)
}

type messagesRepository struct {
	reader  *pgxpool.Pool
	writer  *pgxpool.Pool
	entropy *ulid.MonotonicEntropy
}

func NewMessagesRepository(reader, writer *pgxpool.Pool) MessagesRepository {
	entropy := ulid.Monotonic(rand.Reader, 0)
	return &messagesRepository{
		reader:  reader,
		writer:  writer,
		entropy: entropy,
	}
}

func (r *messagesRepository) Send(ctx context.Context, req models.Message) (string, error) {
	slog.Info("Send message", "chatID", req.ChatID, "userID", req.UserID, "idempotencyKey", req.IdempotencyKey)

	id := ulid.MustNew(ulid.Timestamp(time.Now()), r.entropy)

	query := `INSERT INTO messages (id, idempotency_key, user_id, chat_id, content, status) 
			  VALUES (@id, @idempotency_key, @user_id, @chat_id, @content, @status)`
	args := pgx.NamedArgs{
		"id":              id.String(),
		"idempotency_key": req.IdempotencyKey,
		"user_id":         req.UserID,
		"chat_id":         req.ChatID,
		"content":         req.Body,
		"status":          req.Status,
	}

	_, err := r.writer.Exec(ctx, query, args)
	if err != nil {
		slog.Error("Error sending message", "error", err)
		return id.String(), err
	}

	return id.String(), nil
}

func (r *messagesRepository) List(ctx context.Context, req models.ListMessagesRequest) (models.ListMessagesResponse, error) {
	var (
		messages []models.Message
		total    int32
		page     = req.Page
		limit    = req.Limit
		orderBy  = "created_at DESC"
	)

	if page < 1 {
		page = 1
	}

	if limit < 1 {
		limit = 50
	}

	slog.Info("Listing messages", "chatID", req.ChatID, "page", page, "limit", limit, "orderBy", orderBy)

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		baseQuery := `SELECT m.id, m.idempotency_key, m.user_id, m.chat_id, m.content, m.status, 
						m.created_at, m.updated_at, u.username
					  FROM messages m
					  JOIN users u ON m.user_id = u.id
					  JOIN users_chats uc ON m.chat_id = uc.chat_id
					  WHERE m.chat_id = @chat_id AND uc.user_id = @user_id`

		var whereClause string
		args := pgx.NamedArgs{
			"chat_id": req.ChatID,
			"user_id": req.UserID,
			"limit":   limit,
			"offset":  (page - 1) * limit,
		}

		if req.Before != nil {
			whereClause += " AND m.created_at < @before"
			args["before"] = *req.Before
		}

		if req.After != nil {
			whereClause += " AND m.created_at > @after"
			args["after"] = *req.After
		}

		query := baseQuery + whereClause + " ORDER BY m." + orderBy + " LIMIT @limit OFFSET @offset"

		rows, err := r.reader.Query(ctx, query, args)
		if err != nil {
			slog.Error("Error listing messages", "error", err)
			return err
		}
		defer rows.Close()

		result := []models.Message{}
		for rows.Next() {
			var message models.Message
			var username string
			if err := rows.Scan(
				&message.ID, &message.IdempotencyKey, &message.UserID, &message.ChatID,
				&message.Body, &message.Status, &message.CreatedAt, &message.UpdatedAt,
				&username,
			); err != nil {
				slog.Error("Error scanning message", "error", err)
				return err
			}
			message.User = &models.User{Username: username}
			result = append(result, message)
		}
		if err := rows.Err(); err != nil {
			slog.Error("Error iterating messages", "error", err)
			return err
		}
		messages = result
		return nil
	})

	g.Go(func() error {
		baseQuery := `SELECT COUNT(*)
					  FROM messages m
					  JOIN users_chats uc ON m.chat_id = uc.chat_id
					  WHERE m.chat_id = @chat_id AND uc.user_id = @user_id`

		var whereClause string
		args := pgx.NamedArgs{
			"chat_id": req.ChatID,
			"user_id": req.UserID,
		}

		if req.Before != nil {
			whereClause += " AND m.created_at < @before"
			args["before"] = *req.Before
		}

		if req.After != nil {
			whereClause += " AND m.created_at > @after"
			args["after"] = *req.After
		}

		query := baseQuery + whereClause

		if err := r.reader.QueryRow(ctx, query, args).Scan(&total); err != nil {
			slog.Error("Error counting messages", "error", err)
			return err
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return models.ListMessagesResponse{}, err
	}

	return models.ListMessagesResponse{
		Messages: messages,
		Total:    total,
	}, nil
}

func (r *messagesRepository) Get(ctx context.Context, messageID string) (models.Message, error) {
	slog.Info("Get message", "messageID", messageID)

	query := `SELECT m.id, m.idempotency_key, m.user_id, m.chat_id, m.content, m.status, 
				m.created_at, m.updated_at, u.username
			  FROM messages m
			  JOIN users u ON m.user_id = u.id
			  WHERE m.id = @message_id`
	args := pgx.NamedArgs{
		"message_id": messageID,
	}

	var message models.Message
	var username string
	err := r.reader.QueryRow(ctx, query, args).Scan(
		&message.ID, &message.IdempotencyKey, &message.UserID, &message.ChatID,
		&message.Body, &message.Status, &message.CreatedAt, &message.UpdatedAt,
		&username,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			slog.Info("Message not found", "messageID", messageID)
			return models.Message{}, nil
		}
		slog.Error("Error getting message", "error", err)
		return models.Message{}, err
	}

	message.User = &models.User{Username: username}
	return message, nil
}

func (r *messagesRepository) MarkAsRead(ctx context.Context, messageID, userID string) error {
	slog.Info("Mark message as read", "messageID", messageID, "userID", userID)

	query := `UPDATE messages 
			  SET status = 'READ', updated_at = CURRENT_TIMESTAMP 
			  WHERE id = @message_id 
			  AND chat_id IN (
				  SELECT chat_id FROM users_chats WHERE user_id = @user_id
			  )`
	args := pgx.NamedArgs{
		"message_id": messageID,
		"user_id":    userID,
	}

	result, err := r.writer.Exec(ctx, query, args)
	if err != nil {
		slog.Error("Error marking message as read", "error", err)
		return err
	}

	if result.RowsAffected() == 0 {
		slog.Warn("No message updated - either message doesn't exist or user doesn't have access",
			"messageID", messageID, "userID", userID)
	}

	return nil
}

func (r *messagesRepository) MarkAsDelivered(ctx context.Context, messageID string) error {
	slog.Info("Mark message as delivered", "messageID", messageID)

	query := "UPDATE messages SET status = 'DELIVERED', updated_at = CURRENT_TIMESTAMP WHERE id = @message_id"
	args := pgx.NamedArgs{
		"message_id": messageID,
	}

	_, err := r.writer.Exec(ctx, query, args)
	if err != nil {
		slog.Error("Error marking message as delivered", "error", err)
		return err
	}

	return nil
}

func (r *messagesRepository) GetByIdempotencyKey(ctx context.Context, idempotencyKey string) (models.Message, error) {
	query := `SELECT m.id, m.idempotency_key, m.user_id, m.chat_id, m.content, m.status, 
				m.created_at, m.updated_at, u.username
			  FROM messages m
			  JOIN users u ON m.user_id = u.id
			  WHERE m.idempotency_key = @idempotency_key`
	args := pgx.NamedArgs{
		"idempotency_key": idempotencyKey,
	}

	var message models.Message
	var username string
	err := r.reader.QueryRow(ctx, query, args).Scan(
		&message.ID, &message.IdempotencyKey, &message.UserID, &message.ChatID,
		&message.Body, &message.Status, &message.CreatedAt, &message.UpdatedAt,
		&username,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return models.Message{}, nil
		}
		slog.Error("Error getting message by idempotency key", "error", err)
		return models.Message{}, err
	}

	message.User = &models.User{Username: username}
	return message, nil
}
