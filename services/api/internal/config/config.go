package config

import "os"

type Config struct {
	Addr          string
	DBPath        string
	StaticPath    string
	AppEnv        string
	AdminUsername string
	AdminPassword string
}

func Load() Config {
	return Config{
		Addr:          value("ADDR", ":8080"),
		DBPath:        value("DB_PATH", "./connection-monitor.db"),
		StaticPath:    value("STATIC_PATH", "../../apps/web/dist"),
		AppEnv:        value("APP_ENV", "development"),
		AdminUsername: value("ADMIN_USERNAME", "admin"),
		AdminPassword: value("ADMIN_PASSWORD", "changeme"),
	}
}

func value(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
