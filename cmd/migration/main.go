package main

import (
	"context"
	"flag"

	"github.com/hungdv136/rio/internal/config"
	"github.com/hungdv136/rio/internal/database"
)

func main() {
	ctx := context.Background()
	migrationFile := flag.String("file", "", "name of the migration file to run")
	flag.Parse()

	dbConfig := config.NewDBConfig()
	if err := database.Migrate(ctx, dbConfig, *migrationFile); err != nil {
		panic(err)
	}
}
