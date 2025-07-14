package configs

import (
	"os"
	"strconv"
)

func GetEnv(key, defaultValue string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}
	return value
}

func GetEnvInt(key string, defaultValue int) int {
	valueStr := GetEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}
	val, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return val
}

func GetEnvFloat(key string, defaultValue float64) float64 {
	valueStr := GetEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}
	val, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return defaultValue
	}
	return val
}