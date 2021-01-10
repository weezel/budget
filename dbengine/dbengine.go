package dbengine

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math"
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
	// Descending order regarding the salary
	sort.Slice(debts, func(i, j int) bool {
		return debts[i].Salary < debts[j].Salary
	})

	sumSalaries := float64(debts[1].Salary + debts[0].Salary)
	lesserIncomeRatio := debts[0].Salary / sumSalaries
	greaterIncomeRatio := debts[1].Salary / sumSalaries

	lesserIncomeOwns := debts[1].Expanses * lesserIncomeRatio
	greaterIncomeOwns := debts[0].Expanses * greaterIncomeRatio

	totalExpanses := float64(debts[0].Expanses + debts[1].Expanses)
	expRatioByLesserInc := debts[0].Expanses / totalExpanses
	expRatioByGreaterInc := debts[1].Expanses / totalExpanses

	debt := math.Abs(greaterIncomeOwns - lesserIncomeOwns)

	if expRatioByLesserInc < expRatioByGreaterInc {
		debts[0].Owes = debt
		debts[1].Owes = 0.0
	} else {
		debts[0].Owes = 0.0
		debts[1].Owes = debt
	}
	log.Printf("Debts fetched: %+v", debts)

	return debts, nil
}

func GetMonthlySpending() ([]external.SpendingHistory, error) {
	spending := make([]external.SpendingHistory, 0)

	stmt, err := dbConn.Prepare(MonthlySpendingQuery)
	if err != nil {
		errMsg := fmt.Sprintf(
			"ERROR: Failed to prepare spending query: %v",
			err)
		return []external.SpendingHistory{}, errors.New(errMsg)
	}
	defer stmt.Close()

	res, err := stmt.Query("%" + time.Now().Format("2006"))
	if err != nil {
		errMsg := fmt.Sprintf(
			"ERROR: Couldn't get history of spending: %v",
			err)
		return []external.SpendingHistory{}, errors.New(errMsg)
	}
	defer func() {
		if err := res.Close(); err != nil {
			log.Printf("ERROR: couldn't close file handle in GetMonthlySpending: %s", err)
		}
	}()

	for res.Next() {
		s := external.SpendingHistory{}
		var tmpDate string

		if err := res.Scan(&s.Username, &tmpDate, &s.Spending); err != nil {
			log.Printf("ERROR: couldn't parse spending: %s", err)
			continue
		}

		parsedDate, err := time.Parse("01-2006", tmpDate)
		if err != nil {
			log.Printf("ERROR: Couldn't parse month-year for spending: %v", err)
			continue
		}
		s.MonthYear = parsedDate

		spending = append(spending, s)
	}
	return spending, nil
}

func GetMonthlyPurchasesByUser(username string, month time.Time) ([]external.SpendingHistory, error) {
	spending := make([]external.SpendingHistory, 0)

	stmt, err := dbConn.Prepare(MonthlyPurchasesByUserQuery)
	if err != nil {
		errMsg := fmt.Sprintf(
			"ERROR: Failed to prepare purchases by user query: %v",
			err)
		return []external.SpendingHistory{}, errors.New(errMsg)
	}
	defer func() {
		err := stmt.Close() // FIXME
		if err != nil {
			log.Printf("ERROR: couldn't close purchases by user statement: %s", err)
		}
	}()

	res, err := stmt.Query(username, month.Format("01-2006"))
	if err != nil {
		errMsg := fmt.Sprintf(
			"ERROR: Couldn't get purchases by user: %v",
			err)
		return []external.SpendingHistory{}, errors.New(errMsg)
	}
	defer func() {
		if err := res.Close(); err != nil {
			log.Printf("ERROR: couldn't close file handle in GetMonthlySpending: %s", err)
		}
	}()

	for res.Next() {
		s := external.SpendingHistory{}
		var tmpDate string

		if err := res.Scan(&tmpDate, &s.EventName, &s.Spending); err != nil {
			log.Printf("ERROR: couldn't parse purchases by user: %s", err)
			continue
		}
		parsedDate, err := time.Parse("01-2006", tmpDate)
		if err != nil {
			log.Printf("ERROR: Couldn't parse month-year for spending: %v", err)
			continue
		}
		s.MonthYear = parsedDate

		spending = append(spending, s)
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

	res, err := stmt.Query(
		startMonth.Format("01-2006"),
		endMonth.Format("01-2006"))
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
	log.Printf("Half year salaries starting on %s are %+v",
		startMonth.UTC().Format("01-2006"),
		salaries)

	return salaries, nil
}
