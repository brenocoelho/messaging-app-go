package jwt

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Service interface {
	GenerateToken(user User) (string, error)
	ValidateToken(tokenString string) (*Claims, error)
	RefreshToken(tokenString string) (string, error)
	GetSecretKey() []byte
}

type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Exp      int64  `json:"exp"`
	Iat      int64  `json:"iat"`
}

func (c Claims) GetExpirationTime() (*jwt.NumericDate, error) {
	if c.Exp == 0 {
		return nil, nil
	}
	t := time.Unix(c.Exp, 0)
	return jwt.NewNumericDate(t), nil
}

func (c Claims) GetNotBefore() (*jwt.NumericDate, error) {
	return nil, nil
}

func (c Claims) GetIssuedAt() (*jwt.NumericDate, error) {
	if c.Iat == 0 {
		return nil, nil
	}
	t := time.Unix(c.Iat, 0)
	return jwt.NewNumericDate(t), nil
}

func (c Claims) GetIssuer() (string, error) {
	return "", nil
}

func (c Claims) GetSubject() (string, error) {
	return c.UserID, nil
}

func (c Claims) GetAudience() (jwt.ClaimStrings, error) {
	return nil, nil
}

type jwtService struct {
	secretKey []byte
}

func NewService() Service {
	secretKey := getSecretKey()
	return &jwtService{
		secretKey: secretKey,
	}
}

func getSecretKey() []byte {
	secretKey := os.Getenv("JWT_SECRET_KEY")
	if secretKey != "" {
		return []byte(secretKey)
	}

	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		slog.Error("Failed to generate random secret key", "error", err)
		return []byte("default_jwt_secret_key_change_in_production")
	}

	encodedKey := base64.StdEncoding.EncodeToString(randomBytes)
	slog.Warn("Generated new JWT secret key. Set JWT_SECRET_KEY environment variable for production use", "key", encodedKey[:10]+"...")

	return []byte(encodedKey)
}

func (s *jwtService) GenerateToken(user User) (string, error) {
	now := time.Now()
	expirationTime := now.Add(24 * time.Hour)

	claims := Claims{
		UserID:   user.ID,
		Username: user.Username,
		Email:    user.Email,
		Exp:      expirationTime.Unix(),
		Iat:      now.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.secretKey)
	if err != nil {
		slog.Error("Failed to sign JWT token", "error", err, "userID", user.ID)
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	slog.Info("Generated JWT token", "userID", user.ID, "expiresAt", expirationTime)
	return tokenString, nil
}

func (s *jwtService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.secretKey, nil
	})

	if err != nil {
		slog.Error("Failed to parse JWT token", "error", err)
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	if time.Unix(claims.Exp, 0).Before(time.Now()) {
		return nil, fmt.Errorf("token expired")
	}

	slog.Info("Validated JWT token", "userID", claims.UserID, "username", claims.Username)
	return claims, nil
}

func (s *jwtService) RefreshToken(tokenString string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return s.secretKey, nil
	})

	if err != nil {
		return "", fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return "", fmt.Errorf("invalid token claims")
	}

	expirationTime := time.Unix(claims.Exp, 0)
	if time.Until(expirationTime) > time.Hour {
		return "", fmt.Errorf("token not yet eligible for refresh")
	}

	now := time.Now()
	newExpirationTime := now.Add(24 * time.Hour)

	newClaims := Claims{
		UserID:   claims.UserID,
		Username: claims.Username,
		Email:    claims.Email,
		Exp:      newExpirationTime.Unix(),
		Iat:      now.Unix(),
	}

	newToken := jwt.NewWithClaims(jwt.SigningMethodHS256, newClaims)
	newTokenString, err := newToken.SignedString(s.secretKey)
	if err != nil {
		return "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	slog.Info("Refreshed JWT token", "userID", claims.UserID, "newExpiresAt", newExpirationTime)
	return newTokenString, nil
}

func (s *jwtService) GetSecretKey() []byte {
	return s.secretKey
}
