package users

import (
	"context"
	"crypto/rand"
	"log/slog"
	"time"

	"github.com/brenocoelho/messaging-app-go/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oklog/ulid/v2"
)

type UsersRepository interface {
	Create(ctx context.Context, req models.User) (string, error)
	GetByID(ctx context.Context, id string) (models.User, error)
	GetByEmail(ctx context.Context, email string) (models.User, error)
	Update(ctx context.Context, req models.User) error
}

type usersRepository struct {
	reader  *pgxpool.Pool
	writer  *pgxpool.Pool
	entropy *ulid.MonotonicEntropy
}

func NewUsersRepository(reader, writer *pgxpool.Pool) UsersRepository {
	entropy := ulid.Monotonic(rand.Reader, 0)
	return &usersRepository{
		reader:  reader,
		writer:  writer,
		entropy: entropy,
	}
}

func (r *usersRepository) Create(ctx context.Context, req models.User) (string, error) {
	slog.Info("Create user", "username", req.Username, "email", req.Email)
	id := ulid.MustNew(ulid.Timestamp(time.Now()), r.entropy)

	query := "INSERT INTO users (id, username, email, password_hash) VALUES (@id, @username, @email, @password_hash)"
	args := pgx.NamedArgs{
		"id":            id.String(),
		"username":      req.Username,
		"email":         req.Email,
		"password_hash": req.PasswordHash,
	}
	_, err := r.writer.Exec(ctx, query, args)
	if err != nil {
		slog.Error("Error creating user", "error", err)
		return id.String(), err
	}

	return id.String(), nil
}

func (r *usersRepository) GetByID(ctx context.Context, id string) (models.User, error) {
	slog.Info("Get user by ID", "id", id)

	query := "SELECT id, username, email, password_hash, created_at, updated_at FROM users WHERE id = @id"
	args := pgx.NamedArgs{
		"id": id,
	}

	var user models.User
	err := r.reader.QueryRow(ctx, query, args).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			slog.Info("User not found", "id", id)
			return models.User{}, nil
		}
		slog.Error("Error getting user by ID", "error", err)
		return models.User{}, err
	}

	return user, nil
}

func (r *usersRepository) GetByEmail(ctx context.Context, email string) (models.User, error) {
	slog.Info("Get user by email", "email", email)

	query := "SELECT id, username, email, password_hash, created_at, updated_at FROM users WHERE email = @email"
	args := pgx.NamedArgs{
		"email": email,
	}

	var user models.User
	err := r.reader.QueryRow(ctx, query, args).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			slog.Info("User not found", "email", email)
			return models.User{}, nil
		}
		slog.Error("Error getting user by email", "error", err)
		return models.User{}, err
	}

	return user, nil
}

func (r *usersRepository) Update(ctx context.Context, req models.User) error {
	slog.Info("Update user", "id", req.ID)

	query := "UPDATE users SET username = @username, email = @email, updated_at = CURRENT_TIMESTAMP WHERE id = @id"
	args := pgx.NamedArgs{
		"id":       req.ID,
		"username": req.Username,
		"email":    req.Email,
	}

	_, err := r.writer.Exec(ctx, query, args)
	if err != nil {
		slog.Error("Error updating user", "error", err)
		return err
	}

	return nil
}
