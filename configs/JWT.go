package configs

import (
	"time"
)

type TokenJWT struct {
	JWT                 string
	ExpireDuration      time.Duration
	AdminExpireDuration time.Duration
}

func GetTokenJWTConfig() *TokenJWT {
	jwtSecret := GetEnv("JWT_SECRET", "your-super-secret-key-change-this-in-production")

	expireDurationStr := GetEnv("JWT_EXPIRE_DURATION", "")
	adminExpireDurationStr := GetEnv("JWT_ADMIN_EXPIRE_DURATION", "")

	var expireDuration time.Duration
	var adminExpireDuration time.Duration
	var err error

	if expireDurationStr == "" {
		expireDuration = 30 * 24 * time.Hour // Default 30 hari
	} else {
		expireDuration, err = time.ParseDuration(expireDurationStr)
		if err != nil {
			expireDuration = 30 * 24 * time.Hour
		}
	}

	if adminExpireDurationStr == "" {
		adminExpireDuration = 15 * time.Minute // Default 15 menit untuk admin
	} else {
		adminExpireDuration, err = time.ParseDuration(adminExpireDurationStr)
		if err != nil {
			adminExpireDuration = 15 * time.Minute
		}
	}

	return &TokenJWT{
		JWT:                 jwtSecret,
		ExpireDuration:      expireDuration,
		AdminExpireDuration: adminExpireDuration,
	}
}