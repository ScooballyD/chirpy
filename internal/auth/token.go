package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

func MakeJWT(userID uuid.UUID, tokenSecret string) (string, error) {
	claims := &jwt.RegisteredClaims{
		Issuer:    "chirpy",
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		Subject:   userID.String(),
	}

	tkn := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	jwt, err := tkn.SignedString([]byte(tokenSecret))
	if err != nil {
		return "", fmt.Errorf("error signing token: %v", err)
	}
	return jwt, nil
}

func MakeRefreshToken() (string, error) {
	gen := make([]byte, 10)
	_, err := rand.Read(gen)
	if err != nil {
		return "", fmt.Errorf("unable to creat refresh token: %v", err)
	}

	return hex.EncodeToString(gen), nil
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	tkn, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(tokenSecret), nil
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("unable to parse token: %v", err)
	}

	clms := tkn.Claims.(*jwt.RegisteredClaims)
	id, err := uuid.Parse(clms.Subject)
	if err != nil {
		return uuid.Nil, fmt.Errorf("unable to parse id: %v", err)
	}

	return id, nil
}

func GetBearerToken(headers http.Header) (string, error) {
	Btkn := headers.Get("Authorization")
	Stkn := strings.Split(Btkn, " ")
	if len(Stkn) != 2 {
		return "", fmt.Errorf("bearertoken improper format")
	}

	return strings.TrimSpace(Stkn[1]), nil
}

func GetRefreshToken(headers http.Header) (string, error) {
	Rtkn := headers.Get("Authorization")
	Stkn := strings.Split(Rtkn, " ")
	if len(Stkn) != 2 {
		return "", fmt.Errorf("refresh token improper format")
	}

	return strings.TrimSpace(Stkn[1]), nil
}
