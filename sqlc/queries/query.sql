-- name: AddExpense :exec
INSERT INTO budget(
	username,
	shop_name,
	category,
	price,
	expense_date
) VALUES (
	$1, $2, $3, $4, $5
);

-- name: AddSalary :exec
INSERT INTO salary(
	username,
	salary,
	store_date
) VALUES (
	$1, $2, $3
);

-- name: GetExpenses :many
SELECT username, expense_date, SUM(price)::float AS expenses_sum FROM budget
	GROUP BY expense_date, username
	HAVING expense_date = date_trunc('month', $1)
	ORDER BY username;

-- name: GetSalaryByMonth :one
SELECT salary FROM salary
	WHERE username = $1
		AND store_date = $2
	LIMIT 1;

-- name: GetMonthlySalaries :many
SELECT username, salary, store_date FROM salary
	WHERE store_date BETWEEN date_trunc('month', sqlc.arg('StartMonth')::date)::date
		AND (date_trunc('month', sqlc.arg('EndMonth')::date) + interval '1 month - 1 day')::date
	GROUP BY username, store_date, salary
	ORDER BY username, store_date;

-- name: GetMonthlyExpenses :many
SELECT id, username, expense_date, shop_name, price FROM budget
	GROUP BY username, expense_date, shop_name, price
	HAVING expense_date = $1
	ORDER BY username, expense_date, shop_name, price;

-- name: GetSpendingTimespan :many
SELECT b.username, b.expense_date AS expense_date, SUM(price)::float AS expenses_sum, s.salary
		FROM budget AS b
        INNER JOIN salary AS s ON b.username = s.username
		AND s.store_date = b.expense_date
	WHERE b.expense_date BETWEEN date_trunc('month', sqlc.arg('StartMonth')::date)::date
		AND (date_trunc('month', sqlc.arg('EndMonth')::date) + interval '1 month - 1 day')::date
		AND s.store_date != NULL
		AND b.expense_date != NULL
	GROUP BY b.username, b.expense_date
	ORDER BY b.username, b.expense_date, expenses;

-- const GetSpendingByID :one
SELECT * FROM budget
	WHERE id = $1 AND username = $2;

-- const DeleteSpendingByID :exec
DELETE FROM budget
	WHERE id = $1 AND username = $2;
