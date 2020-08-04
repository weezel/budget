package dbengine

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math"
	"sort"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

/* Yes I know, it's lousy DB design.
Nevertheless, should handle million
rows just fine so we're good.

Why didn't I make it better in the first
hand? Takes time (i.e insertion becomes trickier)
and I wanted to hack it up quickly.
*/
const dbCreationSchema string = `
CREATE TABLE budget(
	id INTEGER PRIMARY KEY,
	username TEXT NOT NULL,
	shopname TEXT NOT NULL,
	category TEXT NOT NULL,
	purchasedate TEXT NOT NULL,
	price REAL NOT NULL
);

CREATE TABLE salary(
	id INTEGER PRIMARY KEY,
	username TEXT NOT NULL,
	salary REAL NOT NULL,
	recordtime TEXT NOT NULL
);`

const insertShoppingQuery string = `
INSERT INTO budget(
	username,
	shopname,
	category,
	purchasedate,
	price
) VALUES (
	:username,
	:shopname,
	:category,
	:purchasedate,
	:price
);`

const insertSalaryQuery string = `
INSERT INTO salary(
	username,
	salary,
	recordtime
) VALUES (
	:username,
	:salary,
	:recordtime
);`

const purchasesQuery string = `
SELECT username, purchasedate, SUM(price) FROM budget
	GROUP BY purchasedate, username
	HAVING purchasedate = ?;
`

const salaryQuery string = `
SELECT salary FROM salary WHERE username = ? AND recordtime = ?;
`

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
	_, err := db.Exec(dbCreationSchema)
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
	stmt, err := dbConn.Prepare(insertSalaryQuery)
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

func InsertShopping(username, shopName, category string, purchaseDate time.Time, price float64) bool {
	stmt, err := dbConn.Prepare(insertShoppingQuery)
	if err != nil {
		log.Printf("ERROR: preparing shopping insert statement failed: %v", err)
		return false
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
		return false
	}
	log.Printf("Wrote %d shopping rows", rowsAffected)
	return true
}

func GetSalaryCompensatedDebts(month time.Time) ([]DebtData, error) {
	debts := make([]DebtData, 0)

	/*
		SELECT username, purchasedate, SUM(price) FROM budget
			GROUP BY purchasedate, username
			HAVING purchasedate = ?;
	*/
	stmt, err := dbConn.Prepare(purchasesQuery)
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
	defer res.Close()

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
		return debts[0].Salary > debts[1].Salary
	})

	greaterExp := math.Max(debts[0].Expanses, debts[1].Expanses)
	lesserExp := math.Min(debts[0].Expanses, debts[1].Expanses)
	pendingDebt := greaterExp - lesserExp
	salaryRatio := debts[0].Salary / debts[1].Salary

	// Person that had greater expanses has already paid the costs and
	// therefore the other party must pay the compensated price
	// (regarding the salary salary) for him/her
	if debts[0].Expanses == lesserExp {
		debts[0].Owes = salaryRatio * pendingDebt
		debts[1].Owes = 0.0
	} else {
		debts[1].Owes = salaryRatio * pendingDebt
		debts[0].Owes = 0.0
	}
	log.Printf("Debts fetched: %+v", debts)

	return debts, nil
}

func getSalaryDataByUser(username string, month time.Time) (float64, error) {
	stmt, err := dbConn.Prepare(salaryQuery)
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
	defer res.Close()

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
