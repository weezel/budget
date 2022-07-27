package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"weezel/budget/confighandler"
	"weezel/budget/dbengine"

	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/pressly/goose"
)

var (
	rollbackAll    bool
	showStatus     bool
	configFilePath string
	wd             string
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
	flag.BoolVar(&showStatus, "s", false, "Show status of migrations")
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

	dbConn, err := dbengine.DBConnForMigrations(conf)
	if err != nil {
		panic(err)
	}
	defer dbConn.Close()

	schemasDir := filepath.Join(wd, "sqlc/schemas")

	if showStatus {
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
