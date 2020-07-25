package dbengine

import (
	"database/sql"
	"log"
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
	purchasedate,
	price
) VALUES (
	:username,
	:shopname,
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

var (
	dbConn *sql.DB
)

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

func InsertShopping(username string, shopName string, purchaseDate time.Time, price float64) bool {
	stmt, err := dbConn.Prepare(insertShoppingQuery)
	if err != nil {
		log.Printf("ERROR: preparing shopping insert statement failed: %v", err)
		return false
	}

	res, err := stmt.Exec(
		sql.Named("username", username),
		sql.Named("shopname", shopName),
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

func GetSalaryCompensatedDebts() {
}
