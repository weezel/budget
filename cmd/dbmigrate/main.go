package main

import (
	"database/sql"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"weezel/budget/confighandler"

	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/pressly/goose"
)

var (
	rollbackAll     bool
	migrationStatus bool
	configFilePath  string
	wd              string
)

func init() {
	log.SetFlags(0)

	var err error
	wd, err = os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	flag.BoolVar(&rollbackAll, "r", false, "Rollback all migrations")
	flag.BoolVar(&migrationStatus, "s", false, "Show status of migrations")
	flag.StringVar(&configFilePath, "f", "budget.toml", "Configuration file")
	flag.Parse()

	configFile, err := ioutil.ReadFile(filepath.Join(wd, configFilePath))
	if err != nil {
		panic(err)
	}
	conf, err := confighandler.LoadConfig(configFile)
	if err != nil {
		panic(err)
	}

	psqlConfig := fmt.Sprintf("user=%s password=%s host=%s port=%s dbname=%s sslmode=disable",
		conf.Postgres.Username,
		conf.Postgres.Password,
		conf.Postgres.Hostname,
		conf.Postgres.Port,
		conf.Postgres.Database)
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
