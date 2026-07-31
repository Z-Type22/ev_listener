package main

import (
	"errors"
	"flag"
	"fmt"
	"sso/internal/config"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	// Config
	cfg := config.MustLoad()

	var migrationsTable, migrationsPath, direction string

	flag.StringVar(&migrationsTable, "migrations-table", "migrations", "name of migrations table")
	flag.StringVar(&migrationsPath, "migrations-path", cfg.MigrationsPath, "path to migrations")
	flag.StringVar(&direction, "direction", "up", "up or down migrations")
	flag.Parse()

	databaseURL := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s&x-migrations-table=%s",
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Name,
		cfg.Database.SSLMode,
		migrationsTable,
	)

	m, err := migrate.New("file://"+migrationsPath, databaseURL)
	if err != nil {
		panic(err)
	}

	switch direction {
	case "up":
		err = m.Up()

	case "down":
		err = m.Down()

	default:
		panic("unknown migration direction: " + direction)
	}

	if err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			fmt.Println("no migrations to apply")
			return
		}

		panic(err)
	}

	fmt.Println("migrations applied")
}
