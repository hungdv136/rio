package config

import (
	"time"

	fs "github.com/hungdv136/rio/internal/storage"
)

const (
	dbConnectionLifetimeSeconds = 300
	dbMaxIdleConnection         = 50
	dbMaxOpenConnection         = 100
	dbUser                      = "admin"
	dbPassword                  = "password"
	dbServer                    = "0.0.0.0:3306"
	dbSchema                    = "rio_services"
)

const (
	serverHost = "0.0.0.0"
	serverPort = "8896"
)

// MySQLConfig contains config data to connect to MySQL database
type MySQLConfig struct {
	DSN                       string `json:"dsn" yaml:"dsn"`
	Server                    string `json:"server" yaml:"server"`
	Schema                    string `json:"schema" yaml:"schema"`
	User                      string `json:"user" yaml:"user"`
	Password                  string `json:"password" yaml:"password"`
	Option                    string `json:"option" yaml:"option"`
	ConnectionLifetimeSeconds int    `json:"connection_lifetime_seconds" yaml:"connection_lifetime_seconds"`
	MaxIdleConnections        int    `json:"max_idle_connections" yaml:"max_idle_connections"`
	MaxOpenConnections        int    `json:"max_open_connections" yaml:"max_open_connections"`
	EnableTracing             bool   `json:"enable_tracing" yaml:"enable_tracing"`
}

// Config defines config for application
type Config struct {
	ServerAddress      string
	FileStorageType    string
	FileStorage        interface{}
	DB                 *MySQLConfig
	StubCacheTTL       time.Duration
	StubCacheStrategy  string
	BodyStoreThreshold int
}

func NewConfig() *Config {
	return &Config{
		ServerAddress:      serverHost + ":" + EV("SERVER_PORT", serverPort),
		DB:                 NewDBConfig(),
		FileStorageType:    getStorageType(),
		FileStorage:        getFileStorageConfig(),
		StubCacheTTL:       EV("STUB_CACHE_TTL", time.Hour),
		StubCacheStrategy:  EV("STUB_CACHE_STRATEGY", "default"),
		BodyStoreThreshold: EV("BODY_STORE_THRESHOLD", 1<<20),
	}
}

// NewDBConfig loads db schema config
func NewDBConfig() *MySQLConfig {
	return &MySQLConfig{
		Server:                    EV("DB_SERVER", dbServer),
		Schema:                    EV("DB_SCHEMA", dbSchema),
		User:                      EV("DB_USER", dbUser),
		Password:                  EV("DB_PASSWORD", dbPassword),
		ConnectionLifetimeSeconds: EV("DB_CONNECTION_LIFETIME_SECONDS", dbConnectionLifetimeSeconds),
		MaxIdleConnections:        EV("DB_MAX_IDLE_CONNECTIONS", dbMaxIdleConnection),
		MaxOpenConnections:        EV("DB_MAX_OPEN_CONNECTIONS", dbMaxOpenConnection),
	}
}

func getFileStorageConfig() interface{} {
	return fs.LocalStorageConfig{
		StoragePath: EV("FILE_DIR", ""),
	}
}

func getStorageType() string {
	return EV("FILE_STORAGE_TYPE", "local")
}

func EV[R any](name string, fallback R) R {
	return fallback
}
