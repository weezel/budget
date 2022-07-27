package dbengine

import (
	"database/sql"
	"fmt"
	"time"
	"weezel/budget/confighandler"
	"weezel/budget/db"

	_ "github.com/jackc/pgx/v4/stdlib"
)

type ExpensesVars struct {
	From      time.Time
	To        time.Time
	Spendings []*db.GetExpensesByTimespanRow
}

type StatisticsVars struct {
	From      time.Time
	To        time.Time
	Spendings []*db.StatisticsByTimespanRow
}

// DBConnForMigrations this connection type is only to be used with database migrations.
func DBConnForMigrations(conf confighandler.TomlConfig) (*sql.DB, error) {
	psqlConfig := fmt.Sprintf("user=%s password=%s host=%s port=%s dbname=%s sslmode=disable",
		conf.Postgres.Username,
		conf.Postgres.Password,
		conf.Postgres.Hostname,
		conf.Postgres.Port,
		conf.Postgres.Database)
	dbConn, err := sql.Open("pgx", psqlConfig)
	if err != nil {
		return nil, err
	}

	return dbConn, nil
}
