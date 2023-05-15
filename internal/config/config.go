package config

import (
	"os"
	"strconv"
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
		ServerAddress:      serverHost + ":" + EVString("SERVER_PORT", serverPort),
		DB:                 NewDBConfig(),
		FileStorageType:    getStorageType(),
		FileStorage:        getFileStorageConfig(),
		StubCacheTTL:       EVDuration("STUB_CACHE_TTL", time.Hour),
		StubCacheStrategy:  EVString("STUB_CACHE_STRATEGY", "default"),
		BodyStoreThreshold: EVInt("BODY_STORE_THRESHOLD", 1<<20),
	}
}

// NewDBConfig loads db schema config
func NewDBConfig() *MySQLConfig {
	return &MySQLConfig{
		Server:                    EVString("DB_SERVER", dbServer),
		Schema:                    EVString("DB_SCHEMA", dbSchema),
		User:                      EVString("DB_USER", dbUser),
		Password:                  EVString("DB_PASSWORD", dbPassword),
		ConnectionLifetimeSeconds: EVInt("DB_CONNECTION_LIFETIME_SECONDS", dbConnectionLifetimeSeconds),
		MaxIdleConnections:        EVInt("DB_MAX_IDLE_CONNECTIONS", dbMaxIdleConnection),
		MaxOpenConnections:        EVInt("DB_MAX_OPEN_CONNECTIONS", dbMaxOpenConnection),
	}
}

func getFileStorageConfig() interface{} {
	return fs.LocalStorageConfig{
		UseTempDir:  true,
		StoragePath: EVString("FILE_DIR", "uploaded_files"),
	}
}

func getStorageType() string {
	return EVString("FILE_STORAGE_TYPE", "local")
}

func EVString(name string, fallback string) string {
	v, ok := os.LookupEnv(name)
	if !ok {
		return fallback
	}

	return v
}

func EVInt(name string, fallback int) int {
	v, ok := os.LookupEnv(name)
	if !ok {
		return fallback
	}

	i, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		panic(err)
	}

	return int(i)
}

func EVDuration(name string, fallback time.Duration) time.Duration {
	v, ok := os.LookupEnv(name)
	if !ok {
		return fallback
	}

	d, err := time.ParseDuration(v)
	if err != nil {
		panic(err)
	}

	return d
}
