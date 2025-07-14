package configs

type DBConfig struct {
	Host     string
	Port     string
	User     string
	DBName   string
	Password string
}

func GetDBConfig() *DBConfig {
	return &DBConfig{
		Host:     GetEnv("DB_HOST", "127.0.0.1"),
		Port:     GetEnv("DB_PORT", "5432"),
		User:     GetEnv("DB_USER", "postgres"),
		Password: GetEnv("DB_PASSWORD", "postgres"),
		DBName:   GetEnv("DB_NAME", "final_project"),
	}
}