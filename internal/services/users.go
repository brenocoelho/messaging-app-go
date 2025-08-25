package services

import (
	"context"
	"errors"
	"log/slog"

	"github.com/brenocoelho/messaging-app-go/internal/models"
	"github.com/brenocoelho/messaging-app-go/internal/repositories/users"
	"github.com/brenocoelho/messaging-app-go/pkg/jwt"
	"golang.org/x/crypto/bcrypt"
)

type UsersService interface {
	CreateUser(ctx context.Context, req models.CreateUserRequest) (models.CreateUserResponse, error)
	Login(ctx context.Context, req models.LoginRequest) (models.LoginResponse, error)
	GetByID(ctx context.Context, userID string) (models.User, error)
}

type usersService struct {
	usersRepo  users.UsersRepository
	jwtService jwt.Service
}

func NewUsersService(usersRepo users.UsersRepository, jwtService jwt.Service) UsersService {
	return &usersService{
		usersRepo:  usersRepo,
		jwtService: jwtService,
	}
}

func (s *usersService) CreateUser(ctx context.Context, req models.CreateUserRequest) (models.CreateUserResponse, error) {
	slog.Info("CreateUser service", "username", req.Username, "email", req.Email)

	existingUser, err := s.usersRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		slog.Error("Error checking if user exists", "error", err)
		return models.CreateUserResponse{}, err
	}

	if existingUser.ID != "" {
		slog.Warn("User already exists", "email", req.Email)
		return models.CreateUserResponse{}, errors.New("user already exists")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("Error hashing password", "error", err)
		return models.CreateUserResponse{}, err
	}

	user := models.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
	}

	userID, err := s.usersRepo.Create(ctx, user)
	if err != nil {
		slog.Error("Error creating user", "error", err)
		return models.CreateUserResponse{}, err
	}

	user.ID = userID

	token, err := s.jwtService.GenerateToken(jwt.User{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
	})
	if err != nil {
		slog.Error("Error generating JWT token", "error", err)
		return models.CreateUserResponse{}, err
	}

	return models.CreateUserResponse{
		Token:  token,
		UserID: user.ID,
	}, nil
}

func (s *usersService) Login(ctx context.Context, req models.LoginRequest) (models.LoginResponse, error) {
	slog.Info("Login service", "email", req.Email)

	user, err := s.usersRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		slog.Error("Error getting user by email", "error", err)
		return models.LoginResponse{}, err
	}

	if user.ID == "" {
		slog.Warn("User not found", "email", req.Email)
		return models.LoginResponse{}, errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		slog.Warn("Invalid password", "email", req.Email)
		return models.LoginResponse{}, errors.New("invalid credentials")
	}

	token, err := s.jwtService.GenerateToken(jwt.User{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
	})
	if err != nil {
		slog.Error("Error generating JWT token", "error", err)
		return models.LoginResponse{}, err
	}

	return models.LoginResponse{
		Token:  token,
		UserID: user.ID,
	}, nil
}

func (s *usersService) GetByID(ctx context.Context, userID string) (models.User, error) {
	slog.Info("GetByID service", "userID", userID)

	return s.usersRepo.GetByID(ctx, userID)
}
