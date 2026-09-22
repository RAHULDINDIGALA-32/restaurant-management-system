package helper

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type SignedDetails struct {
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	UserID    string `json:"user_id"`
	TokenType string `json:"token_type"`
	jwt.RegisteredClaims
}

func getEnv(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func getJWTSecret(tokenType string) string {
	if tokenType == "refresh" {
		return getEnv("JWT_REFRESH_TOKEN_SECRET", "restaurant-refresh-secret")
	}
	return getEnv("JWT_ACCESS_TOKEN_SECRET", "restaurant-access-secret")
}

func getJWTIssuer() string {
	return getEnv("JWT_ISSUER", "restaurant-management-system")
}

func getTokenTTL(tokenType string) time.Duration {
	key := "JWT_ACCESS_TOKEN_TTL"
	fallback := 15 * time.Minute

	if tokenType == "refresh" {
		key = "JWT_REFRESH_TOKEN_TTL"
		fallback = 7 * 24 * time.Hour
	}

	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}

	return duration
}

func generateJTI() (string, error) {
	bytes := make([]byte, 32)

	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes), nil
}

func GenerateAccessToken(
	email string,
	firstName string,
	lastName string,
	userID string,
) (string, error) {
	return generateToken("access", email, firstName, lastName, userID)
}

func GenerateRefreshToken(userID string) (string, error) {
	return generateToken("refresh", "", "", "", userID)
}

func generateToken(
	tokenType string,
	email string,
	firstName string,
	lastName string,
	userID string,
) (string, error) {
	now := time.Now()
	jti, err := generateJTI()
	if err != nil {
		return "", err
	}

	claims := SignedDetails{
		Email:     email,
		FirstName: firstName,
		LastName:  lastName,
		UserID:    userID,
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			Subject:   userID,
			Issuer:    getJWTIssuer(),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(getTokenTTL(tokenType))),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(getJWTSecret(tokenType)))
}

func GenerateAllTokens(
	email string,
	firstName string,
	lastName string,
	userID string,
) (accessToken string, refreshToken string, err error) {
	accessToken, err = GenerateAccessToken(email, firstName, lastName, userID)
	if err != nil {
		return "", "", err
	}

	refreshToken, err = GenerateRefreshToken(userID)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func ValidateAccessToken(tokenString string) (*SignedDetails, error) {
	return validateToken(getJWTSecret("access"), getJWTIssuer(), tokenString, "access")
}

func ValidateRefreshToken(tokenString string) (*SignedDetails, error) {
	return validateToken(getJWTSecret("refresh"), getJWTIssuer(), tokenString, "refresh")
}

func validateToken(
	secret string,
	issuer string,
	tokenString string,
	expectedType string,
) (*SignedDetails, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&SignedDetails{},
		func(token *jwt.Token) (interface{}, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, errors.New("unexpected signing method")
			}
			return []byte(secret), nil
		},
		jwt.WithIssuer(issuer),
	)
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*SignedDetails)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	if claims.TokenType != expectedType {
		return nil, errors.New("invalid token type")
	}

	return claims, nil
}
