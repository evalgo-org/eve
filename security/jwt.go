package security

import (
	"fmt"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

type JWTService struct {
	secret []byte
}

func NewJWTService(secret string) *JWTService {
	return &JWTService{
		secret: []byte(secret),
	}
}

func (j *JWTService) GenerateToken(userID string, expiration time.Duration) (string, error) {
	now := time.Now()

	token, err := jwt.NewBuilder().
		Subject(userID).
		IssuedAt(now).
		Expiration(now.Add(expiration)).
		Build()
	if err != nil {
		return "", fmt.Errorf("failed to build token: %w", err)
	}

	signed, err := jwt.Sign(token, jwt.WithKey(jwa.HS256, j.secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return string(signed), nil
}

func (j *JWTService) ValidateToken(tokenString string) (jwt.Token, error) {
	token, err := jwt.Parse([]byte(tokenString), jwt.WithKey(jwa.HS256, j.secret))
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	return token, nil
}
