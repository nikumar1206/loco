package jwtutil

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type LocoJWTClaims struct {
	UserId           int64  `json:"userId"`
	Username         string `json:"username"`
	ExternalUsername string `json:"externalUsername"`
	jwt.RegisteredClaims
}

const issuer = "loco-api"

var jwtSecret []byte

func init() {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		panic("JWT_SECRET not set")
	}
	jwtSecret = []byte(secret)
}

// GenerateLocoJWT generates a JWT token for Loco API authentication
func GenerateLocoJWT(userID int64, username string, externalUsername string, ttl time.Duration) (string, error) {
	now := time.Now()
	expirationTime := now.Add(ttl)

	claims := &LocoJWTClaims{
		Username:         username,
		UserId:           userID,
		ExternalUsername: externalUsername,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    issuer,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", fmt.Errorf("failed to generate JWT: %w", err)
	}

	return tokenString, nil
}

// ValidateLocoJWT validates a JWT token and returns the claims
func ValidateLocoJWT(tokenString string) (*LocoJWTClaims, error) {
	claims := &LocoJWTClaims{}

	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(issuer),
		jwt.WithLeeway(5*time.Second),
		jwt.WithExpirationRequired(),
	)

	token, err := parser.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		return jwtSecret, nil
	})
	if err != nil {
		slog.Error(err.Error())
		return nil, fmt.Errorf("failed to parse or validate JWT: %w", err)
	}

	if !token.Valid {
		slog.Error("invalid JWT token")
		return nil, fmt.Errorf("invalid JWT token")
	}

	return claims, nil
}
