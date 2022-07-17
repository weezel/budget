package dbengine

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"sort"
	"sync"
	"time"
	"weezel/budget/logger"
	"weezel/budget/utils"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

var (
	once       sync.Once
	db         *sqlx.DB
	dbErr      error
	dbFilePath string
)

type DebtData struct {
	_            struct{}  // Enforces keyed fields
	PurchaseDate time.Time `db:"purchasedate"`
	Expenses     float64   `db:"expenses"`
	Salary       float64   `db:"salary"`
	Username     string    `db:"username"`
	Owes         float64   `db:"owes"`
	SalaryDate   time.Time `db:"recordtime"`
}

type BudgetRow struct {
	_            struct{}
	ID           int64     `db:"id"`
	Price        float64   `db:"price"`
	PurchaseDate time.Time `db:"purchasedate"`
	Username     string    `db:"username"`
	ShopName     string    `db:"shopname"`
	Category     string    `db:"category"`
}

type SpendingHistory struct {
	_         struct{}
	ID        int64     `db:"id"`
	Expenses  float64   `db:"expenses"`
	Salary    float64   `db:"salary"`
	MonthYear time.Time `db:"purchasedate"`
	Username  string    `db:"username"`
	EventName string    `db:"event"`
}

type SpendingHTMLOutput struct {
	From      time.Time
	To        time.Time
	Spendings []SpendingHistory
}

func (d *DebtData) PrettyPrint() string {
	return fmt.Sprintf("%s kulut oli %.4f %s aikana. Palkka tuossa kuussa oli %.2f. Velkaa %.4f",
		d.Username,
		d.Expenses,
		d.PurchaseDate,
		d.Salary,
		d.Owes)
}

// New initializes database once. Also known as singleton.
func New(dataSource string) (*sqlx.DB, error) {
	once.Do(func() {
		if dataSource == ":memory:" {
			db, dbErr = sqlx.Connect("sqlite3", ":memory:")
			if dbErr != nil {
				logger.Fatal(dbErr)
			}
		} else {
			// File based database
			tmp, err := filepath.Abs(filepath.Clean(dataSource))
			if err != nil {
				logger.Panic(err)
			}
			dbFilePath = tmp // Store db file path for later use

			db, dbErr = sqlx.Connect("sqlite3", dbFilePath)
			if dbErr != nil {
				logger.Fatal(dbErr)
			}
		}
		dbErr = db.Ping()
	})

	return db, dbErr
}

func CreateSchema(ctx context.Context) error {
	dirPath := filepath.Dir(dbFilePath)
	exists, err := utils.PathExists(dirPath)
	if err != nil {
		return err
	}
	if exists {
		_, err := db.ExecContext(ctx, DbCreationSchema)
		if err != nil {
			return err
		}
	}
	return nil
}

func GetSpendingRowByID(ctx context.Context, bid int64, username string) (*BudgetRow, error) {
	expense := &BudgetRow{}
	err := db.GetContext(ctx, expense, GetSpendingByIDQuery, bid, username)
	if err != nil {
		return nil, err
	}

	return expense, nil
}

// TODO Return *BudgetRow
func DeleteSpendingByID(ctx context.Context, bid int64, username string) error {
	deletableRow, err := GetSpendingRowByID(ctx, bid, username)
	if err != nil {
		return err
	}

	stmt, err := db.PreparexContext(ctx, DeleteSpendingByIDQuery)
	if err != nil {
		return err
	}
	defer stmt.Close()

	// TODO Use sql.Named?
	res, err := stmt.ExecContext(ctx, bid, username)
	if err != nil {
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected > 0 {
		logger.Infof("User %s deleted the following row from the budget table: %#v",
			username, deletableRow)
	}

	return nil
}

func InsertSalary(ctx context.Context, username string, salary float64, recordTime time.Time) error {
	stmt, err := db.PreparexContext(ctx, InsertSalaryQuery)
	if err != nil {
		return err
	}
	defer stmt.Close()

	res, err := stmt.ExecContext(
		ctx,
		sql.Named("username", username),
		sql.Named("salary", salary),
		sql.Named("recordtime", recordTime.Format("2006-01-02")),
	)
	if err != nil {
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}

	logger.Infof("Wrote %d salary rows", rowsAffected)
	return nil
}

func InsertPurchase(
	ctx context.Context,
	username string,
	shopName string,
	category string,
	purchaseDate time.Time,
	price float64,
) error {
	stmt, err := db.PreparexContext(ctx, InsertShoppingQuery)
	if err != nil {
		return err
	}
	defer stmt.Close()

	res, err := stmt.ExecContext(
		ctx,
		sql.Named("username", username),
		sql.Named("shopname", shopName),
		sql.Named("category", category),
		sql.Named("purchasedate", purchaseDate.Format("2006-01-02")),
		sql.Named("price", price),
	)
	if err != nil {
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}

	logger.Infof("Wrote %d shopping rows", rowsAffected)
	return nil
}

func GetSalaryCompensatedDebts(ctx context.Context, month time.Time) ([]DebtData, error) {
	debts := []DebtData{}
	err := db.SelectContext(ctx, &debts, PurchasesQuery, month.Format("2006-01"))
	if err != nil {
		return nil, err
	}

	if len(debts) < 2 {
		return nil, errors.New("not enough users to calculate debts")
	}

	debts[0].Salary, err = getSalaryDataByUser(ctx, debts[0].Username, month)
	if err != nil {
		return nil, err
	}
	debts[1].Salary, err = getSalaryDataByUser(ctx, debts[1].Username, month)
	if err != nil {
		return nil, err
	}
	// Descending order regarding to salary
	sort.Slice(debts, func(i, j int) bool {
		return debts[i].Salary < debts[j].Salary
	})

	sumSalaries := float64(debts[1].Salary + debts[0].Salary)
	lowerIncomeRatio := debts[0].Salary / sumSalaries
	greaterIncomeRatio := debts[1].Salary / sumSalaries

	lowerIncomeOwes := debts[1].Expenses * lowerIncomeRatio
	greaterIncomeOwes := debts[0].Expenses * greaterIncomeRatio

	totalExpenses := float64(debts[0].Expenses + debts[1].Expenses)
	expRatioByLowerInc := debts[0].Expenses / totalExpenses
	expRatioByGreaterInc := debts[1].Expenses / totalExpenses

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

// Yes, I recognize the functionality is a bit fugly but will fix it later.
func GetMonthlyPurchases(ctx context.Context, startMonth time.Time, endMonth time.Time) (
	[]SpendingHistory,
	error,
) {
	// TODO Use smarter SQL to fetch all data with one shot
	spending := []SpendingHistory{}
	for month := startMonth; month.Before(endMonth) ||
		month.Equal(endMonth); month = month.AddDate(0, 1, 0) {
		s := []SpendingHistory{}
		err := db.SelectContext(ctx, &s, MonthlyPurchasesQuery, month.Format("2006-01-02"))
		if err != nil {
			return nil, err
		}
		spending = append(spending, s...)
	}
	return spending, nil
}

func GetMonthlyData(ctx context.Context, startMonth time.Time, endMonth time.Time) (
	[]SpendingHistory,
	error,
) {
	spending := []SpendingHistory{}
	err := db.SelectContext(ctx, &spending, DateRangeSpendingQuery,
		startMonth.Format("2006-01-02"), endMonth.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}

	return spending, nil
}

func GetSalariesByMonthRange(ctx context.Context, startTime time.Time, endTime time.Time) ([]DebtData, error) {
	salaries := []DebtData{}
	err := db.SelectContext(ctx, &salaries, SalariesQuery,
		startTime.Format("2006-01-02"), endTime.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}

	logger.Debugf("Salaries starting on %s between %s are %+v",
		startTime.Format("2006-01-02"), endTime.Format("2006-01-02"), salaries)

	return salaries, nil
}

func getSalaryDataByUser(ctx context.Context, username string, month time.Time) (float64, error) {
	var salary float64
	err := db.GetContext(ctx, &salary, SalaryQuery, username, month.Format("2006-01"))
	if err != nil {
		return math.NaN(), err
	}

	logger.Debugf("Salary for %s on %s is %.4f", username, month.UTC().Format("01-2006"), salary)

	return salary, nil
}
