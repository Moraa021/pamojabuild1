package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/golang-migrate/migrate/v4"

	"pamojabuild1/backend/db"
	"pamojabuild1/backend/internal/config"
)

const migrationsPath = "db/migrations"

func main() {
	cfg := config.Load()
	runner, err := db.NewMigrator(cfg.DatabaseURL, migrationsPath)
	if err != nil {
		log.Fatal(err)
	}
	defer closeMigrator(runner)

	if err := execute(runner, os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}

func execute(runner *migrate.Migrate, args []string) error {
	if len(args) != 1 && len(args) != 2 {
		return usageError()
	}

	switch args[0] {
	case "up":
		if len(args) == 1 {
			return db.IgnoreNoChange(runner.Up())
		}
		steps, err := positiveSteps(args[1])
		if err != nil {
			return err
		}
		return db.IgnoreNoChange(runner.Steps(steps))
	case "down":
		if len(args) == 1 {
			return errors.New("down requires an explicit number of migrations to roll back")
		}
		steps, err := positiveSteps(args[1])
		if err != nil {
			return err
		}
		return db.IgnoreNoChange(runner.Steps(-steps))
	case "version":
		if len(args) != 1 {
			return usageError()
		}
		version, dirty, err := runner.Version()
		if err != nil {
			return err
		}
		fmt.Printf("version=%d dirty=%t\n", version, dirty)
		return nil
	case "force":
		if len(args) != 2 {
			return errors.New("force requires the version to record")
		}
		version, err := strconv.Atoi(args[1])
		if err != nil || version < 0 {
			return errors.New("force version must be a non-negative integer")
		}
		// Force repairs migration metadata only; it does not execute SQL and
		// should be used after an operator has inspected a dirty migration.
		return runner.Force(version)
	default:
		return usageError()
	}
}

func positiveSteps(raw string) (int, error) {
	steps, err := strconv.Atoi(raw)
	if err != nil || steps <= 0 {
		return 0, errors.New("migration steps must be a positive integer")
	}
	return steps, nil
}

func usageError() error {
	return errors.New("usage: go run ./cmd/migrate <up [steps]|down steps|version|force version>")
}

func closeMigrator(runner *migrate.Migrate) {
	sourceErr, databaseErr := runner.Close()
	if sourceErr != nil {
		log.Printf("close migration source: %v", sourceErr)
	}
	if databaseErr != nil {
		log.Printf("close migration database: %v", databaseErr)
	}
}
