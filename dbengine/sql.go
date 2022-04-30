package dbengine

/* Yes I know, it's a lousy DB design.
Nevertheless, should handle million
rows just fine so we're good.

Why didn't I make it better in the first
hand? Takes time (i.e insertion becomes trickier)
and I wanted to have something functional quickly.
*/
const DbCreationSchema string = `
CREATE TABLE budget(
	id INTEGER PRIMARY KEY,
	username TEXT NOT NULL,
	shopname TEXT NOT NULL,
	category TEXT NOT NULL,
	purchasedate DATE NOT NULL,
	price REAL NOT NULL
);

CREATE TABLE salary(
	id INTEGER PRIMARY KEY,
	username TEXT NOT NULL,
	salary REAL NOT NULL,
	recordtime DATE NOT NULL
);`

const InsertShoppingQuery string = `
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
	date(:purchasedate),
	:price
);`

const InsertSalaryQuery string = `
INSERT INTO salary(
	username,
	salary,
	recordtime
) VALUES (
	:username,
	:salary,
	date(:recordtime)
);`

const PurchasesQuery string = `
SELECT username, purchasedate, SUM(price) FROM budget
	GROUP BY purchasedate, username
	HAVING strftime('%Y-%m', purchasedate) = ?
	ORDER BY username;
`

const SalaryQuery string = `
SELECT salary FROM salary
	WHERE username = ?
	AND strftime('%Y-%m', recordtime) = ?;
`

const SalariesQuery string = `
SELECT username, salary, recordtime FROM salary
	WHERE recordtime BETWEEN ? AND ?
	GROUP BY username, recordtime
	ORDER BY username, recordtime;
`

const MonthlySpendingQuery string = `
SELECT username, purchasedate, SUM(price) FROM budget
	GROUP BY purchasedate, username
	HAVING purchasedate strftime('%Y-%m', purchasedate) = ?
	ORDER BY username, purchasedate;
`
const MonthlyPurchasesQuery string = `
SELECT id, username, strftime('%Y-%m', purchasedate), shopname, price FROM budget
	GROUP BY username, purchasedate, shopname, price
	HAVING strftime('%Y-%m', purchasedate) = ?
	ORDER BY username, purchasedate, shopname, price;
`

const DateRangeSpendingQuery string = `
SELECT b.username, strftime('%Y-%m', b.purchasedate), sum(price) AS expanses, s.salary FROM budget AS b
        LEFT JOIN salary AS s ON b.username = s.username
		AND strftime('%Y-%m-%d', s.recordtime) = strftime('%Y-%m-%d', b.purchasedate)
	WHERE b.purchasedate BETWEEN ? AND DATE(?, 'start of month', '+1 month', '-1 day')
	GROUP BY b.username, b.purchasedate
	ORDER BY b.username, b.purchasedate, expanses;
`

const GetSpendingByIDQuery string = `
SELECT * FROM budget
	WHERE id = ? AND username = ?;
`

const DeleteSpendingByIDQuery string = `
DELETE FROM budget
	WHERE id = ? AND username = ?;
`
