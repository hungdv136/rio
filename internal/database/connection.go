package database

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/hungdv136/rio/internal/config"
	"github.com/hungdv136/rio/internal/log"
	"github.com/uptrace/opentelemetry-go-extra/otelgorm"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	mysqlOption                           = "charset=utf8&parseTime=True&loc=Local&multiStatements=True&maxAllowedPacket=0"
	defaultMySQLConnectionLifetimeSeconds = 300
)

// Connect setups connections to MySQL database
func Connect(ctx context.Context, config *config.MySQLConfig) (*gorm.DB, error) {
	option := config.Option
	if len(option) == 0 {
		option = mysqlOption
	}

	connectionLifetimeSeconds := config.ConnectionLifetimeSeconds
	if connectionLifetimeSeconds == 0 {
		connectionLifetimeSeconds = defaultMySQLConnectionLifetimeSeconds
	}

	source := fmt.Sprintf("%s:%s@tcp(%s)/%s?%s", config.User, config.Password, config.Server, config.Schema, option)
	cfg := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	}

	db, err := gorm.Open(mysql.Open(source), cfg)
	if err != nil {
		log.Error(ctx, "cannot connect to database", config.Schema)
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Error(ctx, "cannot obtain sql database object", config.Schema)
		return nil, err
	}

	sqlDB.SetConnMaxLifetime(time.Duration(connectionLifetimeSeconds) * time.Second)
	sqlDB.SetMaxIdleConns(config.MaxIdleConnections)
	sqlDB.SetMaxOpenConns(config.MaxOpenConnections)

	if config.EnableTracing {
		if err = db.WithContext(ctx).Use(otelgorm.NewPlugin(otelgorm.WithDBName(config.Schema))); err != nil {
			log.Error(ctx, "cannot enable tracing for gorm", config.Schema)
			return nil, err
		}
	}

	log.Info(ctx, "connected to database", config.Schema)
	return db, nil
}

// Disconnect closes the connections to the MySQL database
func Disconnect(ctx context.Context, db *gorm.DB) error {
	sqlDB, err := db.WithContext(ctx).DB()
	if err != nil {
		log.Error(ctx, err)
		return err
	}

	if err := sqlDB.Close(); err != nil {
		log.Error(ctx, "cannot close", err)
		return err
	}

	return nil
}

// ExecuteFileScript runs a specific migration file on a MySQL database using specific path
func ExecuteFileScript(ctx context.Context, db *gorm.DB, filePath string) error {
	migrationSQL, err := os.ReadFile(filePath)
	if err != nil {
		log.Error(ctx, "cannot read sql script", err, filePath)
		return err
	}

	if err := db.WithContext(ctx).Exec(string(migrationSQL)).Error; err != nil {
		log.Error(ctx, err)
		return err
	}

	log.Info(ctx, "executed sql script", filePath)
	return nil
}
