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
	purchasedate TEXT NOT NULL,
	price REAL NOT NULL
);

CREATE TABLE salary(
	id INTEGER PRIMARY KEY,
	username TEXT NOT NULL,
	salary REAL NOT NULL,
	recordtime TEXT NOT NULL
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
	:purchasedate,
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
	:recordtime
);`

const PurchasesQuery string = `
SELECT username, purchasedate, SUM(price) FROM budget
	GROUP BY purchasedate, username
	HAVING purchasedate = ?;
`

const SalaryQuery string = `
SELECT salary FROM salary WHERE username = ? AND recordtime = ?;
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
	HAVING purchasedate LIKE ?
	ORDER BY purchasedate, username;
`
const MonthlyPurchasesByUserQuery string = `
SELECT purchasedate, shopname, price FROM budget
	GROUP BY purchasedate, shopname, price
	HAVING username = ?
	AND purchasedate = ?
	ORDER BY purchasedate, shopname, price;
`

const DateRangeSpendingQuery string = `
SELECT * FROM budget
	WHERE purchasedate BETWEEN ? AND ?
	GROUP BY username, shopname
	ORDER BY purchasedate;
`
