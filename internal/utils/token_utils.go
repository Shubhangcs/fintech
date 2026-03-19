package utils

import (
	cryptorand "crypto/rand"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var paysprintJWTKey = []byte("UTA5U1VEQXdNREF5TXpFMFRucEpORTVFYTNsT2VsbDNUbmM5UFE9PQ==")
var paysprintPartnerID = os.Getenv("PAYSPRINT_PARTNER_ID")

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

// GenerateReqID returns a unique request ID: unix milliseconds + 6 random digits.
func GenerateReqID() string {
	return fmt.Sprintf("%d%06d", time.Now().UnixMilli(), rand.Intn(1000000))
}

// GenerateUUID returns a random UUID v4 string.
func GenerateUUID() string {
	b := make([]byte, 16)
	_, _ = cryptorand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// GeneratePaysprintToken generates a Paysprint JWT for the given reqid.
// Always call GenerateReqID() first and pass the same value to the API payload.
func GeneratePaysprintToken(reqid string) (string, error) {
	claims := jwt.MapClaims{
		"partnerId": paysprintPartnerID,
		"reqid":     reqid,
		"timestamp": time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(paysprintJWTKey)
	if err != nil {
		return "", fmt.Errorf("GeneratePaysprintToken: %w", err)
	}
	return signed, nil
}

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
