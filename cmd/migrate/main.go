// Package main implements database migration utility for the insdr-messenger service.
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

const (
	defaultMigrationsPath = "./migrations"
	defaultMigrateSteps   = 1
)

func main() {
	var (
		migrationsPath string
		databaseURL    string
		steps          int
	)

	flag.StringVar(&migrationsPath, "path", defaultMigrationsPath, "Path to migrations directory")
	flag.IntVar(&steps, "steps", defaultMigrateSteps, "Number of migrations to run")
	flag.Parse()

	databaseURL = os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	// Get command
	args := flag.Args()
	if len(args) == 0 {
		log.Fatal("Please specify a command: up, down, or version")
	}
	command := args[0]

	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			log.Printf("Failed to close database connection: %v", closeErr)
		}
	}()

	// Create postgres driver instance
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		log.Fatalf("Failed to create postgres driver: %v", err)
	}

	// Create migrate instance
	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", migrationsPath), "postgres",
		driver,
	)
	if err != nil {
		log.Fatalf("Failed to create migrate instance: %v", err)
	}

	switch command {
	case "up":
		if err := m.Steps(steps); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("Failed to run migrations up: %v", err)
		}
		version, dirty, err := m.Version()
		if err != nil {
			log.Printf("Error getting migration version: %v", err)
			return
		}
		if dirty {
			log.Printf("WARNING: Database is in dirty state at version %d", version)
		} else {
			log.Printf("Successfully migrated to version %d", version)
		}

	case "down":
		if err := m.Steps(-steps); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("Failed to run migrations down: %v", err)
		}
		version, dirty, err := m.Version()
		if err != nil {
			log.Printf("Error getting migration version: %v", err)
			return
		}
		if dirty {
			log.Printf("WARNING: Database is in dirty state at version %d", version)
		} else if version == 0 {
			log.Println("Successfully rolled back all migrations")
		} else {
			log.Printf("Successfully rolled back to version %d", version)
		}

	case "version":
		version, dirty, err := m.Version()
		if err != nil {
			log.Fatalf("Failed to get version: %v", err)
		}
		if dirty {
			fmt.Printf("Current version: %d (dirty)\n", version)
		} else {
			fmt.Printf("Current version: %d\n", version)
		}

	default:
		log.Fatalf("Unknown command: %s. Use 'up', 'down', or 'version'", command)
	}
}
