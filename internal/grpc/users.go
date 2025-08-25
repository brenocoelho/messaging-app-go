package grpc

import (
	"context"

	"github.com/brenocoelho/messaging-app-go/internal/models"
	"github.com/brenocoelho/messaging-app-go/internal/services"
	pb "github.com/brenocoelho/messaging-app-go/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type UsersGRPCServer struct {
	pb.UnimplementedUsersServiceServer
	usersService services.UsersService
}

func NewUsersGRPCServer(usersService services.UsersService) *UsersGRPCServer {
	return &UsersGRPCServer{
		usersService: usersService,
	}
}

func (s *UsersGRPCServer) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	if req.Username == "" || req.Email == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "username, email, and password are required")
	}

	resp, err := s.usersService.CreateUser(ctx, models.CreateUserRequest{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create user: %v", err)
	}

	user, err := s.usersService.GetByID(ctx, resp.UserID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get created user: %v", err)
	}

	pbUser := &pb.User{
		Id:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		CreatedAt: timestamppb.New(user.CreatedAt),
	}

	return &pb.CreateUserResponse{
		User:  pbUser,
		Token: resp.Token,
	}, nil
}

func (s *UsersGRPCServer) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	if req.Email == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "email and password are required")
	}

	loginResp, err := s.usersService.Login(ctx, models.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid email or password")
	}

	user, err := s.usersService.GetByID(ctx, loginResp.UserID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user details")
	}

	pbUser := &pb.User{
		Id:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		CreatedAt: timestamppb.New(user.CreatedAt),
	}

	return &pb.LoginResponse{
		User:  pbUser,
		Token: loginResp.Token,
	}, nil
}

func (s *UsersGRPCServer) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	user, err := s.usersService.GetByID(ctx, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user: %v", err)
	}

	if user.ID == "" {
		return nil, status.Error(codes.NotFound, "user not found")
	}

	pbUser := &pb.User{
		Id:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		CreatedAt: timestamppb.New(user.CreatedAt),
	}

	return &pb.GetUserResponse{
		User: pbUser,
	}, nil
}
