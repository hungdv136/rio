package database

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/hungdv136/rio"
	"github.com/hungdv136/rio/internal/config"
)

func TestMain(m *testing.M) {
	ctx := context.Background()
	dbConfig := config.NewDBConfig()
	gormDB, err := Connect(ctx, dbConfig)
	if err != nil {
		panic(err)
	}

	if err := ExecuteFileScript(ctx, gormDB, "../../schema/reset_db.sql"); err != nil {
		panic(err)
	}

	if err := Migrate(ctx, dbConfig, "../../schema/migration"); err != nil {
		panic(err)
	}

	store, err := NewStubDBStore(ctx, dbConfig)
	if err != nil {
		panic(err)
	}

	resetOption := &rio.ResetQueryOption{
		Namespace: uuid.NewString(),
		Tag:       uuid.NewString(),
	}

	if err := store.Reset(ctx, resetOption); err != nil {
		panic(err)
	}

	code := m.Run()
	os.Exit(code)
}
