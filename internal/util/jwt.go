package util

import (
	"time"

	"portolio-backend/configs"

	"github.com/dgrijalva/jwt-go"
)

func GenerateJWTToken(userID uint, userRole string) (string, error) {
	tokenJWT := configs.GetTokenJWTConfig()
	secretKey := []byte(tokenJWT.JWT)

	claims := jwt.MapClaims{
		"ID":   userID,
		"role": userRole,
		"exp":  time.Now().Add(tokenJWT.ExpireDuration).Unix(),
		"iat":  time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(secretKey)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func GenerateAdminSessionToken(userID uint, userRole string, expireDuration time.Duration, IPAddress string, UserAgent string) (string, error) {
	tokenJWT := configs.GetTokenJWTConfig()
	secretKey := []byte(tokenJWT.JWT)

	claims := jwt.MapClaims{
		"ID":   userID,
		"role": userRole,
		"exp":  time.Now().Add(expireDuration).Unix(),
		"iat":  time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(secretKey)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}