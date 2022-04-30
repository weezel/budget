package dbengine

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"reflect"
	"sort"
	"time"
	"weezel/budget/external"
	"weezel/budget/logger"

	_ "github.com/mattn/go-sqlite3"
)

var (
	dbConn *sql.DB
)

type DebtData struct {
	Date     time.Time
	Username string
	Expanses float64
	Owes     float64
	Salary   float64
}

type BudgetRow struct {
	Purchasedate time.Time
	Username     string
	Shopname     string
	Category     string
	ID           int64
	Price        float64
}

func (d *DebtData) PrettyPrint() string {
	return fmt.Sprintf("%s kulut oli %.4f %s aikana. Palkka tuossa kuussa oli %.2f. Velkaa %.4f",
		d.Username,
		d.Expanses,
		d.Date,
		d.Salary,
		d.Owes)
}

func CreateSchema(db *sql.DB) {
	_, err := db.Exec(DbCreationSchema)
	if err != nil {
		logger.Fatal(err)
	}
}

func UpdateDBReference(db *sql.DB) {
	if db == nil {
		return
	}
	dbConn = db
}

func GetSpendingRowByID(bid int64, username string) (BudgetRow, error) {
	stmt, err := dbConn.Prepare(GetSpendingByIDQuery)
	if err != nil {
		return BudgetRow{}, err
	}
	defer stmt.Close()

	budgetRow := BudgetRow{}
	err = stmt.QueryRow(bid, username).Scan(
		&budgetRow.ID,
		&budgetRow.Username,
		&budgetRow.Shopname,
		&budgetRow.Category,
		&budgetRow.Purchasedate,
		&budgetRow.Price)
	if err != nil {
		return BudgetRow{}, err
	} else if err == nil && reflect.DeepEqual(budgetRow, BudgetRow{}) {
		errMsg := fmt.Sprintf("User %s not permitted to delete row %d from the budget table",
			username, bid)
		return BudgetRow{}, errors.New(errMsg)
	}

	return budgetRow, nil
}

func DeleteSpendingByID(bid int64, username string) error {
	deletableRow, err := GetSpendingRowByID(bid, username)
	if err != nil {
		return err
	}

	stmt, err := dbConn.Prepare(DeleteSpendingByIDQuery)
	if err != nil {
		return err
	}
	defer stmt.Close()

	res, err := stmt.Exec(bid, username)
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

func InsertSalary(username string, salary float64, recordTime time.Time) bool {
	stmt, err := dbConn.Prepare(InsertSalaryQuery)
	if err != nil {
		logger.Errorf("preparing salary insert statement failed: %v", err)
		return false
	}

	res, err := stmt.Exec(
		sql.Named("username", username),
		sql.Named("salary", salary),
		sql.Named("recordtime", recordTime.Format("2006-01-02")),
	)
	if err != nil {
		logger.Errorf("failed to insert salary data: %s", err)
		return false
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		logger.Errorf("getting salary rows failed: %v", err)
		return false
	}
	logger.Infof("Wrote %d salary rows", rowsAffected)
	return true
}

func InsertPurchase(
	username string,
	shopName string,
	category string,
	purchaseDate time.Time,
	price float64,
) error {
	stmt, err := dbConn.Prepare(InsertShoppingQuery)
	if err != nil {
		logger.Errorf("preparing shopping insert statement failed: %v", err)
		return errors.New("Virhe, ei onnistuttu yhdistämään kantaan")
	}

	res, err := stmt.Exec(
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
		logger.Errorf("getting shopping rows failed: %v", err)
		return errors.New("Virhe, ei saatu ostosdataa")
	}
	logger.Infof("Wrote %d shopping rows", rowsAffected)
	return nil
}

func GetSalaryCompensatedDebts(month time.Time) ([]DebtData, error) {
	debts := make([]DebtData, 0)

	stmt, err := dbConn.Prepare(PurchasesQuery)
	if err != nil {
		return []DebtData{}, err
	}
	defer stmt.Close()

	res, err := stmt.Query(month.Format("2006-01"))
	if err != nil {
		return []DebtData{}, err
	}
	defer func() {
		if err := res.Close(); err != nil {
			logger.Errorf("couldn't close file handle in GetSalaryCompensatedDebts: %s", err)
		}
	}()

	for res.Next() {
		d := DebtData{}
		err = res.Scan(&d.Username, &d.Date, &d.Expanses)
		if err != nil {
			return []DebtData{}, err
		}
		debts = append(debts, d)
	}

	if len(debts) < 2 {
		return []DebtData{}, err
	}

	debts[0].Salary, err = getSalaryDataByUser(debts[0].Username, month)
	if err != nil {
		return []DebtData{}, err
	}
	debts[1].Salary, err = getSalaryDataByUser(debts[1].Username, month)
	if err != nil {
		return []DebtData{}, err
	}
	// Descending order regarding to salary
	sort.Slice(debts, func(i, j int) bool {
		return debts[i].Salary < debts[j].Salary
	})

	sumSalaries := float64(debts[1].Salary + debts[0].Salary)
	lowerIncomeRatio := debts[0].Salary / sumSalaries
	greaterIncomeRatio := debts[1].Salary / sumSalaries

	lowerIncomeOwes := debts[1].Expanses * lowerIncomeRatio
	greaterIncomeOwes := debts[0].Expanses * greaterIncomeRatio

	totalExpanses := float64(debts[0].Expanses + debts[1].Expanses)
	expRatioByLowerInc := debts[0].Expanses / totalExpanses
	expRatioByGreaterInc := debts[1].Expanses / totalExpanses

	debt := math.Abs(greaterIncomeOwes - lowerIncomeOwes)

	logger.Debugf("Sum of salaries: %.2f", sumSalaries)
	logger.Debugf("Lower income ration: %.2f", lowerIncomeRatio)
	logger.Debugf("Lower income owes: %.2f", lowerIncomeOwes)
	logger.Debugf("Greater income ration: %.2f", greaterIncomeRatio)
	logger.Debugf("Greater income owes: %.2f", greaterIncomeOwes)
	logger.Debugf("Expanses ratio by lower income: %.2f", expRatioByLowerInc)
	logger.Debugf("Expanses ratio by greater income: %.2f", expRatioByGreaterInc)
	logger.Debugf("Total expanses: %.2f", totalExpanses)
	logger.Debugf("Debt in the end: %.2f", debt)

	if expRatioByLowerInc < expRatioByGreaterInc {
		debts[0].Owes = debt
		debts[1].Owes = 0.0
	} else {
		debts[0].Owes = 0.0
		debts[1].Owes = debt
	}
	logger.Infof("Debts fetched: %#v", debts)

	return debts, nil
}

// Yes, I recognize the functionality is a bit fugly but will fix it later.
func GetMonthlyPurchases(startMonth time.Time, endMonth time.Time) (
	[]external.SpendingHistory,
	error,
) {
	var spending []external.SpendingHistory = []external.SpendingHistory{}

	stmt, err := dbConn.Prepare(MonthlyPurchasesQuery)
	if err != nil {
		return []external.SpendingHistory{}, err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			logger.Errorf("couldn't close purchases by user statement: %s", err)
		}
	}()

	for iterMonth := startMonth; iterMonth.Before(endMonth) || iterMonth.Equal(endMonth); iterMonth = iterMonth.AddDate(0, 1, 0) {
		res, err := stmt.Query(iterMonth.Format("2006-01"))
		if err != nil {
			return []external.SpendingHistory{}, err
		}
		defer func() {
			if err := res.Close(); err != nil {
				logger.Errorf("couldn't close file handle in GetMonthlyPurchasesByUser: %s", err)
			}
		}()

		for res.Next() {
			s := external.SpendingHistory{}

			var tmpDate string
			err = res.Scan(&s.ID, &s.Username, &tmpDate, &s.EventName, &s.Spending)
			if err != nil {
				logger.Errorf("couldn't parse purchases by user: %s", err)
				continue
			}
			s.MonthYear, err = time.Parse("2006-01", tmpDate)
			if err != nil {
				logger.Error(err)
				continue
			}

			spending = append(spending, s)
		}
	}
	return spending, nil
}

func GetMonthlyData(startMonth time.Time, endMonth time.Time) (
	[]external.SpendingHistory,
	error,
) {
	var spending []external.SpendingHistory = []external.SpendingHistory{}

	stmt, err := dbConn.Prepare(DateRangeSpendingQuery)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to prepare purchases by user query: %v", err)
		return []external.SpendingHistory{}, errors.New(errMsg)
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			logger.Errorf("Couldn't close data gathering by user statement: %s", err)
		}
	}()

	res, err := stmt.Query(
		startMonth.Format("2006-01-02"),
		endMonth.Format("2006-01-02"))
	if err != nil {
		errMsg := fmt.Sprintf("Couldn't get monthly data: %v", err)
		return []external.SpendingHistory{}, errors.New(errMsg)
	}
	defer func() {
		if err := res.Close(); err != nil {
			logger.Errorf("couldn't close file handle: %s", err)
		}
	}()

	for res.Next() {
		s := external.SpendingHistory{}
		var spendingTmp sql.NullFloat64
		var salaryTmp sql.NullFloat64

		var t string
		if err := res.Scan(&s.Username, &t, &spendingTmp, &salaryTmp); err != nil {
			logger.Errorf("couldn't parse purchases for user: %s", err)
			continue
		}

		s.MonthYear, err = time.Parse("2006-01", t)
		if err != nil {
			logger.Error(err)
			continue
		}

		if spendingTmp.Valid {
			s.Spending = spendingTmp.Float64
		} else {
			s.Spending = math.NaN()
		}

		if salaryTmp.Valid {
			s.Salary = salaryTmp.Float64
		} else {
			s.Salary = math.NaN()
		}

		spending = append(spending, s)
	}

	return spending, nil
}

func getSalaryDataByUser(username string, month time.Time) (float64, error) {
	stmt, err := dbConn.Prepare(SalaryQuery)
	if err != nil {
		return math.NaN(), err
	}
	defer stmt.Close()

	res, err := stmt.Query(username, month.Format("2006-01"))
	if err != nil {
		return math.NaN(), err
	}
	defer func() {
		if err := res.Close(); err != nil {
			logger.Errorf("couldn't close file handle in getSalaryDataByUser: %s", err)
		}
	}()

	var salary float64
	for res.Next() {
		err = res.Scan(&salary)
		if err != nil {
			return math.NaN(), err
		}
	}
	logger.Infof("Salary for %s on %s is %.4f",
		username,
		month.UTC().Format("01-2006"),
		salary)

	return salary, nil
}

func GetSalariesByMonthRange(startMonth time.Time, endMonth time.Time) ([]DebtData, error) {
	stmt, err := dbConn.Prepare(SalariesQuery)
	if err != nil {
		return []DebtData{}, err
	}
	defer stmt.Close()

	s, e := startMonth.UTC().Format("2006-01"),
		endMonth.UTC().Format("2006-01")
	res, err := stmt.Query(s, e)
	if err != nil {
		return []DebtData{}, err
	}
	defer func() {
		if err := res.Close(); err != nil {
			logger.Errorf("couldn't close file handle in GetSalariesByMonth: %s", err)
		}
	}()

	var salaries []DebtData = make([]DebtData, 0)
	for res.Next() {
		var salary DebtData
		err = res.Scan(&salary.Username, &salary.Salary, &salary.Date)
		if err != nil {
			return []DebtData{}, err
		}
		if salary.Salary > 0 {
			salary.Salary = 1.0
		}
		salaries = append(salaries, salary)

	}
	logger.Debugf("Salaries starting on %s between %s are %+v",
		s, e, salaries)

	return salaries, nil
}
