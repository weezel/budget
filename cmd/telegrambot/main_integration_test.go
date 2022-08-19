//go:build integration
// +build integration

package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"
	"weezel/budget/confighandler"
	"weezel/budget/db"
	"weezel/budget/dbengine"
	"weezel/budget/logger"
	"weezel/budget/outputs"
	"weezel/budget/shortlivedpage"
	"weezel/budget/utils"

	"github.com/google/go-cmp/cmp"
	"github.com/jackc/pgx/v4/pgxpool"
)

var (
	wd   string
	conn *pgxpool.Pool
)

func init() {
	wd, _ = os.Getwd()
	ctx := context.Background()

	configFileName := "../../integrations.toml"
	configFile, err := ioutil.ReadFile(filepath.Join(wd, configFileName))
	if err != nil {
		panic(fmt.Errorf(">1> %s", err))
	}
	conf, err := confighandler.LoadConfig(configFile)
	if err != nil {
		panic(fmt.Errorf(">2> %s", err))
	}

	// Perform database migrations
	err = dbMigrations(conf)
	if err != nil {
		panic(fmt.Errorf(">3> %s", err))
	}

	conn, err = dbengine.New(context.Background(), conf.Postgres)
	if err != nil {
		panic(fmt.Errorf(">4> %s", err))
	}

	_, err = conn.Exec(ctx, "DELETE FROM budget_schema.expense;")
	if err != nil {
		panic(fmt.Errorf(">5> %s", err))
	}
	_, err = conn.Exec(ctx, "DELETE FROM budget_schema.salary;")
	if err != nil {
		panic(fmt.Errorf(">6> %s", err))
	}

	addContent()
}

func generateStatsHTMLPage(
	ctx context.Context,
	startMonth time.Time,
	endMonth time.Time,
) ([]byte, error) {
	stats, err := dbengine.StatisticsByTimespan(ctx, startMonth, endMonth)
	if err != nil {
		return nil, err
	}

	detailedExpenses, err := dbengine.GetExpensesByTimespan(ctx, startMonth, endMonth)
	if err != nil {
		return nil, err
	}

	htmlPage, err := outputs.RenderStatsHTML(outputs.StatisticsVars{
		From:       startMonth,
		To:         endMonth,
		Statistics: stats,
		Detailed:   detailedExpenses,
	})
	if err != nil {
		return nil, err
	}
	htmlPageHash := utils.CalcSha256Sum(htmlPage)
	shortlivedPage := shortlivedpage.ShortLivedPage{
		TTLSeconds: 600,
		StartTime:  time.Now(),
		HTMLPage:   &htmlPage,
	}
	// If hash already exits, Add function returns false.
	if ok := shortlivedpage.Add(htmlPageHash, shortlivedPage); ok {
		endTime := shortlivedPage.StartTime.Add(
			time.Duration(shortlivedPage.TTLSeconds))
		logger.Infof("Added shortlived data page %s with end time %s",
			htmlPageHash, endTime)
	}

	return htmlPage, nil
}

func addContent() {
	bdb := db.New(conn)

	for i := 1; i < 11; i++ {
		// Expense
		_, err := bdb.AddExpense(context.Background(), db.AddExpenseParams{
			Username:    "Jorma",
			ShopName:    "Lidl",
			Category:    "Groceries",
			Price:       float64(i) * 2,
			ExpenseDate: time.Date(2020, 4, i, 1, 0, 0, 0, time.UTC),
		})
		if err != nil {
			panic(err)
		}
		_, err = bdb.AddExpense(context.Background(), db.AddExpenseParams{
			Username:    "Jorma",
			ShopName:    "Beer",
			Category:    "Leisure",
			Price:       float64(i) * 2,
			ExpenseDate: time.Date(2020, 8, i, 1, 0, 0, 0, time.UTC),
		})
		if err != nil {
			panic(err)
		}

		_, err = bdb.AddExpense(context.Background(), db.AddExpenseParams{
			Username:    "Alice",
			ShopName:    "IceHockery",
			Category:    "Sports",
			Price:       float64(i) * 3.14,
			ExpenseDate: time.Date(2020, 4, 10+i, 1, 1, 0, 0, time.UTC),
		})
		if err != nil {
			panic(err)
		}

		// Salary
		_, err = bdb.AddSalary(context.Background(), db.AddSalaryParams{
			Username:  "Jorma",
			Salary:    1000.37,
			StoreDate: time.Date(2020, time.Month(i), 1, 1, 0, 0, 0, time.UTC),
		})
		if err != nil {
			panic(err)
		}
		_, err = bdb.AddSalary(context.Background(), db.AddSalaryParams{
			Username:  "Alice",
			Salary:    1788.12,
			StoreDate: time.Date(2020, time.Month(i), 1, 1, 0, 0, 0, time.UTC),
		})
		if err != nil {
			panic(err)
		}
		_, err = bdb.AddSalary(context.Background(), db.AddSalaryParams{
			Username:  "Alice",
			Salary:    1788.12,
			StoreDate: time.Date(2021, time.Month(i)+1, 1, 1, 0, 0, 0, time.UTC),
		})
		if err != nil {
			panic(err)
		}
	}
}

// TestIntegration_main is imitating end to end test without Telegram being involved.
func TestIntegration_main(t *testing.T) {
	if testing.Short() {
		t.Skipf("Skipping integration test %s due `short` was defined", t.Name())
	}

	ctx := context.Background()

	shortlivedpage.InitScheduler()

	startMonth := utils.GetDate([]string{"01-2020"}, "01-2006")
	endMonth := utils.GetDate([]string{"06-2020"}, "01-2006")

	statsPage, err := generateStatsHTMLPage(ctx, startMonth, endMonth)
	if err != nil {
		t.Error(err)
	}
	// os.WriteFile("stats-out.html", statsPage, 0o600) // For observation
	expectedPage, err := os.ReadFile("./stats-test_expected.html")
	if err != nil {
		panic(err)
	}

	if diff := cmp.Diff(expectedPage, statsPage); diff != "" {
		t.Fatalf("%s: Stats HTML page differs from the expected one:\n%s",
			t.Name(), diff)
	}
}
