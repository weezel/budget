package main

/*
This program is self-contained and ephemeral. Intention is to provide
a tool for a SQLite -> PostgreSQL migration and remove the tool after
a short while.

Currently tool can be used like this:
	go run cmd/sqlite2postgres/main.go

By default it expects SQLite file to be named `budget.db` and `.env`
variables configured for PostgreSQL. Different database file can be
passed with `-f` flag.
Once migration is done, it prints "Migration completed" (we're omtiting
sqlite.c related warnings here).

NOTE: Currently database doesn't prevent adding the same named event
twice with the same sum, so be sure to run it only once! This is a
deliberate decision.

I didn't want to scatter related structs and variables to different files
and wanted to keep them in one place, since when file is going to be deleted,
deleting one file will be enough. Bear with me.
*/

import (
	"context"
	"flag"
	"log"
	"os"
	"path/filepath"
	"time"
	"weezel/budget/db"
	"weezel/budget/dbengine"

	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3" // This is intended (silents revive)
)

var (
	wd           string
	sqliteDBPath string
	envFilePath  string
)

type BudgetRow struct {
	_            struct{} // Enforces keyed fields
	ID           int64    `db:"id"`
	Username     string   `db:"username"`
	ShopName     string   `db:"shopname"`
	Category     string   `db:"category"`
	PurchaseDate string   `db:"purchasedate"`
	Price        float64  `db:"price"`
}

type SalaryRow struct {
	_          struct{}
	ID         int64   `db:"id"`
	Username   string  `db:"username"`
	Salary     float64 `db:"salary"`
	RecordTime string  `db:"recordtime"`
}

func ParseTime(tt string) time.Time {
	t, err := time.Parse("2006-01-02", tt)
	if err != nil {
		log.Println("ERR: " + err.Error())
		t = time.Time{}
	}
	return t
}

func init() {
	log.SetFlags(0)

	var err error
	wd, err = os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
}

func initSQLiteConnection(ctx context.Context, dataSource string) (*sqlx.DB, error) {
	dbFilePath, err := filepath.Abs(filepath.Clean(dataSource))
	if err != nil {
		return nil, err
	}
	sqliteDB, err := sqlx.Connect("sqlite3", dbFilePath)
	if err != nil {
		return nil, err
	}

	return sqliteDB, sqliteDB.Ping()
}

func main() {
	ctx := context.Background()

	flag.StringVar(&sqliteDBPath, "d", "budget.db", "SQLite database path")
	flag.StringVar(&envFilePath, "e", ".", ".env file path")
	flag.Parse()

	if sqliteDBPath == "" {
		flag.Usage()
		os.Exit(1)
	}

	postgresDB, err := dbengine.InitPostgresConnection(ctx, envFilePath)
	if err != nil {
		panic(err)
	}
	defer postgresDB.Close()
	budgetDB := db.New(postgresDB)

	sqliteDB, err := initSQLiteConnection(ctx, sqliteDBPath)
	if err != nil {
		panic(err)
	}

	// Get salaries
	salaries := []SalaryRow{}
	err = sqliteDB.SelectContext(ctx, &salaries, "SELECT * FROM salary;")
	if err != nil {
		panic(err)
	}

	// Get budget
	expenses := []BudgetRow{}
	err = sqliteDB.SelectContext(ctx, &expenses, "SELECT * FROM budget;")
	if err != nil {
		panic(err)
	}

	// Insert salaries to Postgres
	for _, s := range salaries {
		_, err = budgetDB.AddSalary(ctx, db.AddSalaryParams{
			Username:  s.Username,
			Salary:    s.Salary,
			StoreDate: ParseTime(s.RecordTime),
		})
		if err != nil {
			panic(err)
		}
	}
	// Insert expenses to Postgres
	for _, b := range expenses {
		_, err = budgetDB.AddExpense(ctx, db.AddExpenseParams{
			Username:    b.Username,
			ShopName:    b.ShopName,
			Category:    b.Category,
			Price:       b.Price,
			ExpenseDate: ParseTime(b.PurchaseDate),
		})
		if err != nil {
			panic(err)
		}
	}

	log.Println("Migration completed")
}
