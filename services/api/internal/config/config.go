package config

import "os"

type Config struct {
	Addr                    string
	DBPath                  string
	StaticPath              string
	AppEnv                  string
	AdminUsername           string
	AdminPassword           string
	AuthCookieSecure        bool
	EventOutboxDir          string
	DSMNotificationsEnabled bool
}

func Load() Config {
	return Config{
		Addr:                    value("ADDR", ":8080"),
		DBPath:                  value("DB_PATH", "./connection-monitor.db"),
		StaticPath:              value("STATIC_PATH", "../../apps/web/dist"),
		AppEnv:                  value("APP_ENV", "development"),
		AdminUsername:           os.Getenv("ADMIN_USERNAME"),
		AdminPassword:           os.Getenv("ADMIN_PASSWORD"),
		AuthCookieSecure:        boolValue("AUTH_COOKIE_SECURE", false),
		EventOutboxDir:          os.Getenv("EVENT_OUTBOX_DIR"),
		DSMNotificationsEnabled: boolValue("DSM_NOTIFICATIONS_ENABLED", false),
	}
}

func value(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func boolValue(key string, fallback bool) bool {
	switch os.Getenv(key) {
	case "1", "true", "TRUE", "yes", "YES", "on", "ON":
		return true
	case "0", "false", "FALSE", "no", "NO", "off", "OFF":
		return false
	default:
		return fallback
	}
}
