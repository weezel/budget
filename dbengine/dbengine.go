package dbengine

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math"
	"reflect"
	"sort"
	"time"
	"weezel/budget/external"

	_ "github.com/mattn/go-sqlite3"
)

var (
	dbConn *sql.DB
)

type DebtData struct {
	Username string
	Expanses float64
	Owes     float64
	Salary   float64
	Date     string
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
		log.Fatal(err)
	}
}

func UpdateDBReference(db *sql.DB) {
	if db == nil {
		return
	}
	dbConn = db
}

type BudgetRow struct {
	ID           int64
	Username     string
	Shopname     string
	Category     string
	Purchasedate string
	Price        float64
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
		errMsg := fmt.Sprintf("User not permitted to delete row %d from budget table",
			bid)
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
		log.Printf("Deleted the following row from budget table: %#v", deletableRow)
	}

	return nil
}

func InsertSalary(username string, salary float64, recordTime time.Time) bool {
	stmt, err := dbConn.Prepare(InsertSalaryQuery)
	if err != nil {
		log.Printf("ERROR: preparing salary insert statement failed: %v", err)
		return false
	}

	res, err := stmt.Exec(
		sql.Named("username", username),
		sql.Named("salary", salary),
		sql.Named("recordtime", recordTime.Format("01-2006")),
	)
	if err != nil {
		log.Printf("ERROR: failed to insert salary data: %s", err)
		return false
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		log.Printf("ERROR: getting salary rows failed: %v", err)
		return false
	}
	log.Printf("Wrote %d salary rows", rowsAffected)
	return true
}

func InsertShopping(username, shopName, category string, purchaseDate time.Time, price float64) error {
	stmt, err := dbConn.Prepare(InsertShoppingQuery)
	if err != nil {
		log.Printf("ERROR: preparing shopping insert statement failed: %v", err)
		return errors.New("Virhe, ei onnistuttu yhdistämään kantaan")
	}

	res, err := stmt.Exec(
		sql.Named("username", username),
		sql.Named("shopname", shopName),
		sql.Named("category", category),
		sql.Named("purchasedate", purchaseDate.Format("01-2006")),
		sql.Named("price", price),
	)
	if err != nil {
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		log.Printf("ERROR: getting shopping rows failed: %v", err)
		return errors.New("Virhe, ei saatu ostosdataa")
	}
	log.Printf("Wrote %d shopping rows", rowsAffected)
	return nil
}

func GetSalaryCompensatedDebts(month time.Time) ([]DebtData, error) {
	debts := make([]DebtData, 0)

	stmt, err := dbConn.Prepare(PurchasesQuery)
	if err != nil {
		errMsg := fmt.Sprintf("ERROR: Failed to prepare purchase query: %v", err)
		return []DebtData{}, errors.New(errMsg)
	}
	defer stmt.Close()

	res, err := stmt.Query(month.Format("01-2006"))
	if err != nil {
		errMsg := fmt.Sprintf("ERROR: couldn't get purchase data: %v", err)
		return []DebtData{}, errors.New(errMsg)
	}
	defer func() {
		if err := res.Close(); err != nil {
			log.Printf("ERROR: couldn't close file handle in GetSalaryCompensatedDebts: %s", err)
		}
	}()

	for res.Next() {
		d := DebtData{}
		err = res.Scan(&d.Username, &d.Date, &d.Expanses)
		if err != nil {
			errMsg := fmt.Sprintf("ERROR: Couldn't assign debt data: %v", err)
			return []DebtData{}, errors.New(errMsg)
		}
		debts = append(debts, d)
	}

	if len(debts) < 2 {
		errMsg := "ERROR: Someone didn't spend at all on this month"
		return []DebtData{}, errors.New(errMsg)
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

	log.Printf("Sum of salaries: %.2f", sumSalaries)
	log.Printf("Lower income ration: %.2f", lowerIncomeRatio)
	log.Printf("Lower income owes: %.2f", lowerIncomeOwes)
	log.Printf("Greater income ration: %.2f", greaterIncomeRatio)
	log.Printf("Greater income owes: %.2f", greaterIncomeOwes)
	log.Printf("Expanses ratio by lower income: %.2f", expRatioByLowerInc)
	log.Printf("Expanses ratio by greater income: %.2f", expRatioByGreaterInc)
	log.Printf("Total expanses: %.2f", totalExpanses)
	log.Printf("Debt in the end: %.2f", debt)

	if expRatioByLowerInc < expRatioByGreaterInc {
		debts[0].Owes = debt
		debts[1].Owes = 0.0
	} else {
		debts[0].Owes = 0.0
		debts[1].Owes = debt
	}
	log.Printf("Debts fetched: %+v", debts)

	return debts, nil
}

// Yes, I recognize the functionality is a bit fugly but will fix it later.
func GetMonthlyPurchasesByUser(username string, startMonth time.Time, endMonth time.Time) (
	map[time.Time][]external.SpendingHistory,
	error,
) {
	spending := make(map[time.Time][]external.SpendingHistory)

	stmt, err := dbConn.Prepare(MonthlyPurchasesByUserQuery)
	if err != nil {
		errMsg := fmt.Sprintf(
			"ERROR: Failed to prepare purchases by user query: %v",
			err)
		return map[time.Time][]external.SpendingHistory{}, errors.New(errMsg)
	}
	defer func() {
		err := stmt.Close() // FIXME
		if err != nil {
			log.Printf("ERROR: couldn't close purchases by user statement: %s", err)
		}
	}()

	for iterMonth := startMonth; iterMonth.Before(endMonth) || iterMonth.Equal(endMonth); iterMonth = iterMonth.AddDate(0, 1, 0) {
		res, err := stmt.Query(username, iterMonth.Format("01-2006"))
		if err != nil {
			errMsg := fmt.Sprintf(
				"ERROR: Couldn't get purchases by user: %v",
				err)
			return map[time.Time][]external.SpendingHistory{}, errors.New(errMsg)
		}
		defer func() {
			if err := res.Close(); err != nil {
				log.Printf("ERROR: couldn't close file handle in GetMonthlyPurchasesByUser: %s", err)
			}
		}()

		for res.Next() {
			s := external.SpendingHistory{}
			var tmpDate string

			if err := res.Scan(&s.ID, &tmpDate, &s.EventName, &s.Spending); err != nil {
				log.Printf("ERROR: couldn't parse purchases by user: %s", err)
				continue
			}
			parsedDate, err := time.Parse("01-2006", tmpDate)
			if err != nil {
				log.Printf("ERROR: Couldn't parse month-year for spending: %v", err)
				continue
			}
			s.MonthYear = parsedDate

			spending[s.MonthYear] = append(spending[s.MonthYear], s)
		}
	}
	return spending, nil
}

func GetMonthlyData(startMonth time.Time, endMonth time.Time) (
	map[time.Time][]external.SpendingHistory,
	error,
) {
	spending := make(map[time.Time][]external.SpendingHistory)

	stmt, err := dbConn.Prepare(DateRangeSpendingQuery)
	if err != nil {
		errMsg := fmt.Sprintf(
			"ERROR: Failed to prepare purchases by user query: %v",
			err)
		return map[time.Time][]external.SpendingHistory{}, errors.New(errMsg)
	}
	defer func() {
		err := stmt.Close() // FIXME
		if err != nil {
			log.Printf("couldn't close data gathering by user statement: %s", err)
		}
	}()

	res, err := stmt.Query(
		startMonth.Format("01-2006"),
		endMonth.Format("01-2006"))
	if err != nil {
		errMsg := fmt.Sprintf(
			"ERROR: Couldn't get monthly data: %v",
			err)
		return map[time.Time][]external.SpendingHistory{}, errors.New(errMsg)
	}
	defer func() {
		if err := res.Close(); err != nil {
			log.Printf("ERROR: couldn't close file handle: %s", err)
		}
	}()

	for res.Next() {
		s := external.SpendingHistory{}
		var tmpDate string
		var spendingTmp sql.NullFloat64
		var salaryTmp sql.NullFloat64

		if err := res.Scan(&s.Username, &tmpDate, &spendingTmp, &salaryTmp); err != nil {
			log.Printf("ERROR: couldn't parse purchases by user: %s", err)
			continue
		}

		parsedDate, err := time.Parse("01-2006", tmpDate)
		if err != nil {
			log.Printf("ERROR: couldn't parse month-year: %v", err)
			continue
		}
		s.MonthYear = parsedDate

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

		spending[s.MonthYear] = append(spending[s.MonthYear], s)
	}

	return spending, nil
}

func getSalaryDataByUser(username string, month time.Time) (float64, error) {
	stmt, err := dbConn.Prepare(SalaryQuery)
	if err != nil {
		errMsg := fmt.Sprintf(
			"ERROR: Failed to prepare salary query: %v",
			err)
		return math.NaN(), errors.New(errMsg)
	}
	defer stmt.Close()

	res, err := stmt.Query(username, month.Format("01-2006"))
	if err != nil {
		errMsg := fmt.Sprintf(
			"ERROR: Couldn't get list of salaries: %v",
			err)
		return math.NaN(), errors.New(errMsg)
	}
	defer func() {
		if err := res.Close(); err != nil {
			log.Printf("ERROR: couldn't close file handle in getSalaryDataByUser: %s", err)
		}
	}()

	var salary float64
	for res.Next() {
		err = res.Scan(&salary)
		if err != nil {
			errMsg := fmt.Sprintf("ERROR: Couldn't assign salary data: %v", err)
			return math.NaN(), errors.New(errMsg)
		}
	}
	log.Printf("Salary for %s on %s is %.4f",
		username,
		month.UTC().Format("01-2006"),
		salary)

	return salary, nil
}

func GetSalariesByMonthRange(startMonth time.Time, endMonth time.Time) ([]DebtData, error) {
	stmt, err := dbConn.Prepare(SalariesQuery)
	if err != nil {
		errMsg := fmt.Sprintf(
			"ERROR: Failed to prepare salary query: %v",
			err)
		return []DebtData{}, errors.New(errMsg)
	}
	defer stmt.Close()

	s, e := startMonth.UTC().Format("01-2006"),
		endMonth.UTC().Format("01-2006")
	res, err := stmt.Query(s, e)
	if err != nil {
		errMsg := fmt.Sprintf(
			"ERROR: Couldn't get list of salaries: %v",
			err)
		return []DebtData{}, errors.New(errMsg)
	}
	defer func() {
		if err := res.Close(); err != nil {
			log.Printf("ERROR: couldn't close file handle in GetSalariesByMonth: %s", err)
		}
	}()

	var salaries []DebtData = make([]DebtData, 0)
	for res.Next() {
		var salary DebtData
		err = res.Scan(&salary.Username, &salary.Salary, &salary.Date)
		if err != nil {
			errMsg := fmt.Sprintf("ERROR: Couldn't assign salary data: %v", err)
			return []DebtData{}, errors.New(errMsg)
		}
		if salary.Salary > 0 {
			salary.Salary = 1.0
		}
		salaries = append(salaries, salary)

	}
	log.Printf("Salaries starting on %s between %s are %+v",
		s, e, salaries)

	return salaries, nil
}
