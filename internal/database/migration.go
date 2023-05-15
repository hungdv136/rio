package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4"
	"github.com/hungdv136/rio/internal/config"
	"github.com/hungdv136/rio/internal/log"

	// blank import
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func Migrate(ctx context.Context, config *config.MySQLConfig, dir string) error {
	connectionString := fmt.Sprintf("mysql://%s", resolveDatabaseConnectionURL(config))
	m, err := migrate.New(fmt.Sprintf("file://%s", dir), connectionString)
	if err != nil {
		return err
	}

	defer m.Close()

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log.Info(ctx, "no migration needed")
			return nil
		}

		return err
	}

	return nil
}

func resolveDatabaseConnectionURL(config *config.MySQLConfig) string {
	format := mysql.Config{
		User:                 config.User,
		Passwd:               config.Password,
		Addr:                 config.Server,
		Net:                  "tcp",
		DBName:               config.Schema,
		ParseTime:            true,
		MultiStatements:      true,
		Loc:                  time.Local,
		AllowNativePasswords: true,
	}
	return format.FormatDSN()
}
