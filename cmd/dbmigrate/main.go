package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/golobby/dotenv"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/pressly/goose"
)

var (
	rollbackAll     bool
	migrationStatus bool
	wd              string
)

type dbConfig struct {
	username string `env:"DB_USERNAME"`
	password string `env:"DB_PASSWORD"`
	hostname string `env:"DB_HOST"`
	port     string `env:"DB_PORT"`
	dbName   string `env:"DB_NAME"`
}

func init() {
	log.SetFlags(0)

	var err error
	wd, err = os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
}

func validateEnvVars(dbConf dbConfig) {
	if dbConf.username == "" {
		log.Fatal("Missing DB_USERNAME")
	}
	if dbConf.password == "" {
		log.Fatal("Missing DB_PASSWORD")
	}
	if dbConf.hostname == "" {
		log.Fatal("Missing DB_HOST")
	}
	if dbConf.port == "" {
		log.Fatal("Missing DB_PORT")
	}
}

func main() {
	flag.BoolVar(&rollbackAll, "r", false, "Rollback all migrations")
	flag.BoolVar(&migrationStatus, "s", false, "Show status of migrations")
	flag.Parse()

	dbConf := dbConfig{}
	fhandle, err := os.Open(".env")
	if err != nil {
		panic(err)
	}
	err = dotenv.NewDecoder(fhandle).Decode(&dbConf)
	if err != nil {
		panic(err)
	}
	validateEnvVars(dbConf)

	psqlConfig := fmt.Sprintf("user=%s password=%s host=%s port=%s dbname=%s sslmode=disable",
		dbConf.username, dbConf.password, dbConf.hostname, dbConf.port, dbConf.dbName)
	dbConn, err := sql.Open("pgx", psqlConfig)
	if err != nil {
		log.Fatal(err)
	}
	defer dbConn.Close()

	schemasDir := filepath.Join(wd, "sqlc/schemas")

	if migrationStatus {
		if err = goose.Status(dbConn, schemasDir); err != nil {
			fmt.Println(err)
		}
		return
	}

	if rollbackAll {
		log.Println("Rollback the database migrations")
		// Rollback all the migrations until they are gone
		for {
			if err = goose.Down(dbConn, schemasDir); err != nil {
				log.Printf("error while rolling back: %s\n", err)
				break
			}
		}
		fmt.Println("Rollbacks completed")
	} else {
		// Do the DB Migrations
		if err := goose.Status(dbConn, schemasDir); err != nil {
			fmt.Println(err)
			return
		}
		if err := goose.Up(dbConn, schemasDir); err != nil {
			fmt.Println(err)
			return
		}
	}
	log.Println("Database migration completed")
}
