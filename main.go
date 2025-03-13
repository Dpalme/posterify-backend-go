package main

import (
	"log"
	"os"
	"strconv"

	pg "github.com/Dpalme/posterify-backend/postgres"
	"github.com/Dpalme/posterify-backend/server"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/joho/godotenv/autoload"
	_ "github.com/lib/pq"
)

type config struct {
	port           string
	dbURI          string
	migrationSteps uint
}

func main() {
	cfg := envConfig()

	db, err := pg.Open(cfg.dbURI)
	if err != nil {
		log.Fatalf("cannot open database: %v", err)
	}
	driver, err := postgres.WithInstance(db.DB.DB, &postgres.Config{})
	if err != nil {
		log.Fatalf("cannot run migrations: %v", err)
	}
	m, err := migrate.NewWithDatabaseInstance(
		"file://./postgres/migrations",
		"postgres", driver)
	if err != nil {
		log.Fatalf("cannot run migrations: %v", err)
	}
	err = m.Up()
	if err != nil {
		if err.Error() != "no change" {
			log.Fatalf("error running migrations: %v", err)
		} else {
			log.Print("no migrations to run")
		}
	}

	srv := server.NewServer(db)
	log.Fatal(srv.Run(cfg.port))
}

func envConfig() config {
	port, ok := os.LookupEnv("PORT")

	if !ok {
		panic("PORT not provided")
	}

	dbURI, ok := os.LookupEnv("POSTGRESQL_URL")

	if !ok {
		panic("POSTGRESQL_URL not provided")
	}

	migrationStepsString, ok := os.LookupEnv("MIGRATION_VERSION")

	if !ok {
		panic("MIGRATION_VERSION not provided")
	}

	migrationSteps, err := strconv.ParseUint(migrationStepsString, 10, 32)
	if err != nil {
		panic("MIGRATION_VERSION not a number")
	}

	return config{port, dbURI, uint(migrationSteps)}
}
