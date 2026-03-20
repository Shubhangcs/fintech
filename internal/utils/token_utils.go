package utils

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	PaysprintJWTKey    = []byte(os.Getenv("PAYSPRINT_JWT_KEY"))
	PaysprintPartnerID = os.Getenv("PAYSPRINT_PARTNER_ID")
)

// Paysprint Token Generation

func GeneratePaysprintToken(reqid string) (string, error) {
	claims := jwt.MapClaims{
		"partnerId": PaysprintPartnerID,
		"reqid":     reqid,
		"timestamp": time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(PaysprintJWTKey)
	if err != nil {
		return "", fmt.Errorf("GeneratePaysprintToken: %w", err)
	}
	return signed, nil
}

// Server Token Generation

type TokenClaims struct {
	UserID   string `json:"user_id"`
	UserName string `json:"user_name"`
	jwt.RegisteredClaims
}

var (
	secretKey   = os.Getenv("JWT_SECRET_KEY")
	expiry      = time.Hour * 24
	tokenIssuer = os.Getenv("JWT_TOKEN_ISSUER")
)

func GenerateToken(userID, userName string) (string, error) {
	claims := TokenClaims{
		UserID:   userID,
		UserName: userName,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    tokenIssuer,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secretKey))
}

func ValidateToken(tokenString string) (*TokenClaims, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&TokenClaims{},
		func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("invalid signing method")
			}
			return []byte(secretKey), nil
		},
	)
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*TokenClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid or expired token")
	}

	if claims.Issuer != tokenIssuer {
		return nil, errors.New("invalid token issuer")
	}

	return claims, nil
}
