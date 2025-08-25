package jwt

import (
	"context"
	"fmt"
	"strings"

	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type contextKey string

const (
	userIDKey    contextKey = "user_id"
	usernameKey  contextKey = "username"
	emailKey     contextKey = "email"
	jwtClaimsKey contextKey = "jwt_claims"
)

type Interceptor struct {
	jwtService Service
}

func NewInterceptor(jwtService Service) *Interceptor {
	return &Interceptor{
		jwtService: jwtService,
	}
}

func (i *Interceptor) UnaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	if shouldSkipAuth(info.FullMethod) {
		return handler(ctx, req)
	}

	claims, err := i.extractAndValidateToken(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "authentication failed: %v", err)
	}

	ctx = context.WithValue(ctx, userIDKey, claims.UserID)
	ctx = context.WithValue(ctx, usernameKey, claims.Username)
	ctx = context.WithValue(ctx, emailKey, claims.Email)
	ctx = context.WithValue(ctx, jwtClaimsKey, claims)

	slog.Info("JWT authentication successful", "userID", claims.UserID, "username", claims.Username, "method", info.FullMethod)

	return handler(ctx, req)
}

func (i *Interceptor) StreamInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	if shouldSkipAuth(info.FullMethod) {
		return handler(srv, ss)
	}

	claims, err := i.extractAndValidateToken(ss.Context())
	if err != nil {
		return status.Errorf(codes.Unauthenticated, "authentication failed: %v", err)
	}

	ctx := context.WithValue(ss.Context(), userIDKey, claims.UserID)
	ctx = context.WithValue(ctx, usernameKey, claims.Username)
	ctx = context.WithValue(ctx, emailKey, claims.Email)
	ctx = context.WithValue(ctx, jwtClaimsKey, claims)

	wrappedStream := &wrappedServerStream{
		ServerStream: ss,
		ctx:          ctx,
	}

	slog.Info("JWT authentication successful for stream", "userID", claims.UserID, "username", claims.Username, "method", info.FullMethod)

	return handler(srv, wrappedStream)
}

func (i *Interceptor) extractAndValidateToken(ctx context.Context) (*Claims, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, fmt.Errorf("no metadata found")
	}

	authHeaders := md.Get("authorization")
	if len(authHeaders) == 0 {
		return nil, fmt.Errorf("no authorization header found")
	}

	authHeader := authHeaders[0]
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return nil, fmt.Errorf("invalid authorization header format")
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == "" {
		return nil, fmt.Errorf("empty token")
	}

	claims, err := i.jwtService.ValidateToken(tokenString)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	return claims, nil
}

func shouldSkipAuth(method string) bool {
	skipMethods := []string{
		"/messaging.UsersService/CreateUser",
		"/messaging.UsersService/Login",
	}

	for _, skipMethod := range skipMethods {
		if method == skipMethod {
			return true
		}
	}

	return false
}

type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}

// GetUserFromContext extracts user information from context
func GetUserFromContext(ctx context.Context) (string, string, string, error) {
	userID, ok := ctx.Value(userIDKey).(string)
	if !ok || userID == "" {
		// Fallback to old contextKey type for backward compatibility
		userID, ok = ctx.Value(contextKey("user_id")).(string)
		if !ok || userID == "" {
			return "", "", "", fmt.Errorf("user_id not found in context")
		}
	}

	username, ok := ctx.Value(usernameKey).(string)
	if !ok {
		username, ok = ctx.Value(contextKey("username")).(string)
		if !ok {
			return "", "", "", fmt.Errorf("username not found in context")
		}
	}

	email, ok := ctx.Value(emailKey).(string)
	if !ok {
		email, ok = ctx.Value(contextKey("email")).(string)
		if !ok {
			return "", "", "", fmt.Errorf("email not found in context")
		}
	}

	return userID, username, email, nil
}

// GetClaimsFromContext extracts JWT claims from context
func GetClaimsFromContext(ctx context.Context) (*Claims, error) {
	claims, ok := ctx.Value(jwtClaimsKey).(*Claims)
	if !ok {
		// Fallback to old contextKey type for backward compatibility
		claims, ok = ctx.Value(contextKey("jwt_claims")).(*Claims)
		if !ok {
			return nil, fmt.Errorf("jwt_claims not found in context")
		}
	}
	return claims, nil
}
