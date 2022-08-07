package dbengine

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
	"weezel/budget/confighandler"
	"weezel/budget/db"
	"weezel/budget/logger"

	"github.com/jackc/pgx/v4/pgxpool"
)

const (
	dbConnRetries = 3
)

var (
	once   sync.Once
	dbPool *pgxpool.Pool
	dbErr  error
)

// New initializes database once. Also known as singleton.
func New(ctx context.Context, dbConf confighandler.Postgres) (*pgxpool.Pool, error) {
	once.Do(func() {
		// TODO Use unix-socket
		pgConfigURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			dbConf.Username, dbConf.Password, dbConf.Hostname, dbConf.Port, dbConf.Database)
		dbPool, dbErr = pgxpool.Connect(ctx, pgConfigURL)
		if dbErr != nil {
			logger.Fatal(dbErr)
		}
	})

	retries := 0
	started := time.Now()
	for {
		if dbErr = dbPool.Ping(ctx); dbErr == nil {
			break
		}
		delay := math.Ceil(math.Pow(2, float64(retries)))
		time.Sleep(time.Duration(delay) * time.Second)
		retries++

		logger.Infof("Retrying db connection %d/%d (%s since started)",
			retries, dbConnRetries, time.Since(started))

		if retries > dbConnRetries {
			return nil, fmt.Errorf("Couldn't connect to database after %d retries", retries-1)
		}
	}

	return dbPool, dbErr
}

func AddExpense(
	ctx context.Context,
	username string,
	shopName string,
	category string,
	expenseDate time.Time,
	price float64,
) (int32, error) {
	bdb := db.New(dbPool)
	return bdb.AddExpense(ctx, db.AddExpenseParams{
		Username:    username,
		ShopName:    shopName,
		Category:    category,
		Price:       price,
		ExpenseDate: expenseDate,
	})
}

func DeleteExpenseByID(ctx context.Context, bid int32, username string) (*db.BudgetSchemaExpense, error) {
	bdb := db.New(dbPool)
	return bdb.DeleteExpenseByID(ctx, db.DeleteExpenseByIDParams{
		ID:       bid,
		Username: username,
	})
}

func GetAggrExpensesByTimespan(
	ctx context.Context,
	startTime,
	endTime time.Time,
) ([]*db.GetAggrExpensesByTimespanRow, error) {
	bdb := db.New(dbPool)
	return bdb.GetAggrExpensesByTimespan(ctx, db.GetAggrExpensesByTimespanParams{
		StartTime: startTime,
		EndTime:   endTime,
	})
}

func GetExpensesByTimespan(ctx context.Context, startTime, endTime time.Time) ([]*db.GetExpensesByTimespanRow, error) {
	bdb := db.New(dbPool)
	return bdb.GetExpensesByTimespan(ctx, db.GetExpensesByTimespanParams{
		StartTime: startTime,
		EndTime:   endTime,
	})
}

func AddSalary(ctx context.Context, username string, salary float64, storeDate time.Time) (int32, error) {
	bdb := db.New(dbPool)
	return bdb.AddSalary(ctx, db.AddSalaryParams{
		Username:  username,
		Salary:    salary,
		StoreDate: storeDate,
	})
}

func DeleteSalaryByID(ctx context.Context, id int32, username string) (*db.BudgetSchemaSalary, error) {
	bdb := db.New(dbPool)
	return bdb.DeleteSalaryByID(ctx, db.DeleteSalaryByIDParams{
		ID:       id,
		Username: username,
	})
}

func GetUserSalaryByMonth(ctx context.Context, username string, month time.Time) (float64, error) {
	bdb := db.New(dbPool)
	return bdb.GetUserSalaryByMonth(ctx, db.GetUserSalaryByMonthParams{
		Username: username,
		Month:    month,
	})
}

func GetSalariesByTimespan(
	ctx context.Context,
	startTime time.Time,
	endTime time.Time,
) ([]*db.GetSalariesByTimespanRow, error) {
	bdb := db.New(dbPool)
	return bdb.GetSalariesByTimespan(ctx, db.GetSalariesByTimespanParams{
		StartTime: startTime,
		EndTime:   endTime,
	})
}

func StatisticsByTimespan(
	ctx context.Context,
	startTime time.Time,
	endTime time.Time,
) ([]*db.StatisticsAggrByTimespanRow, error) {
	bdb := db.New(dbPool)
	return bdb.StatisticsAggrByTimespan(ctx, db.StatisticsAggrByTimespanParams{
		StartTime: startTime,
		EndTime:   endTime,
	})
}

func GetSalaryCompensatedDebts(
	ctx context.Context,
	startTime time.Time,
	endTime time.Time,
) ([]*db.StatisticsAggrByTimespanRow, error) {
	bdb := db.New(dbPool)
	debts, err := bdb.StatisticsAggrByTimespan(ctx, db.StatisticsAggrByTimespanParams{
		StartTime: startTime,
		EndTime:   endTime,
	})
	if err != nil {
		return nil, err
	}

	if len(debts) != 2 {
		return nil, fmt.Errorf("Not enough, or too much users to calculate debts (was %d)",
			len(debts))
	}

	// Descending order regarding to salary
	sort.Slice(debts, func(i, j int) bool {
		return debts[i].Salary < debts[j].Salary
	})

	sumSalaries := float64(debts[1].Salary + debts[0].Salary)
	lowerIncomeRatio := debts[0].Salary / sumSalaries
	greaterIncomeRatio := debts[1].Salary / sumSalaries

	lowerIncomeOwes := debts[1].ExpensesSum * lowerIncomeRatio
	greaterIncomeOwes := debts[0].ExpensesSum * greaterIncomeRatio

	totalExpenses := float64(debts[0].ExpensesSum + debts[1].ExpensesSum)
	expRatioByLowerInc := debts[0].ExpensesSum / totalExpenses
	expRatioByGreaterInc := debts[1].ExpensesSum / totalExpenses

	debt := math.Abs(greaterIncomeOwes - lowerIncomeOwes)

	logger.Debugf("Sum of salaries: %.2f", sumSalaries)
	logger.Debugf("Lower income ration: %.2f", lowerIncomeRatio)
	logger.Debugf("Lower income owes: %.2f", lowerIncomeOwes)
	logger.Debugf("Greater income ration: %.2f", greaterIncomeRatio)
	logger.Debugf("Greater income owes: %.2f", greaterIncomeOwes)
	logger.Debugf("Expenses ratio by lower income: %.2f", expRatioByLowerInc)
	logger.Debugf("Expenses ratio by greater income: %.2f", expRatioByGreaterInc)
	logger.Debugf("Total expenses: %.2f", totalExpenses)
	logger.Debugf("Debt in the end: %.2f", debt)

	if expRatioByLowerInc < expRatioByGreaterInc {
		debts[0].Owes = debt
		debts[1].Owes = 0.0
	} else {
		debts[0].Owes = 0.0
		debts[1].Owes = debt
	}
	logger.Debugf("Debts fetched: %#v", debts)

	return debts, nil
}
