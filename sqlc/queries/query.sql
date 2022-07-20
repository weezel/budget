--
-- Expenses
--

-- name: AddExpense :one
INSERT INTO budget_schema.expense(
	username,
	shop_name,
	category,
	price,
	expense_date
) VALUES ($1, $2, $3, $4, $5) RETURNING id;


-- name: DeleteExpenseByID :one
DELETE FROM budget_schema.expense
	WHERE id = $1 AND username = $2
	RETURNING *;

-- name: GetExpensesByTimespan :many
SELECT id, username, expense_date, shop_name, price FROM budget_schema.expense
	WHERE expense_date BETWEEN sqlc.arg('start_time')::date
		AND sqlc.arg('end_time')::date + interval '1 month - 1 day'
	ORDER BY username, expense_date, shop_name, price;

-- name: GetAggrExpensesByTimespan :many
SELECT username, date_trunc('month', expense_date)::date AS months, SUM(price)::float AS expenses_sum
	FROM budget_schema.expense
	WHERE expense_date BETWEEN sqlc.arg('start_time')::date
		AND sqlc.arg('end_time')::date + interval '1 month - 1 day'
	GROUP BY username, months, shop_name
	ORDER BY months, username;

--
-- Salaries
--

-- name: AddSalary :one
INSERT INTO budget_schema.salary(username, salary, store_date)
	VALUES($1, $2, $3) RETURNING id;

-- name: DeleteSalaryByID :one
DELETE FROM budget_schema.salary
	WHERE id = $1 AND username = $2
	RETURNING *;

-- name: GetUserSalaryByMonth :one
SELECT salary FROM budget_schema.salary
	WHERE username = $1
	AND store_date = date_trunc('month', sqlc.arg('month')::date);

-- name: GetSalariesByTimespan :many
SELECT username, salary, date_trunc('month', store_date)::date AS months FROM budget_schema.salary
	WHERE store_date BETWEEN date_trunc('month', sqlc.arg('start_time')::date)::date
		AND date_trunc('month', sqlc.arg('end_time')::date)::date + interval '1 month - 1 day'
	GROUP BY username, months, salary
	ORDER BY username, months;

--
-- Miscellaneous
--

-- name: StatisticsAggrByTimespan :many
SELECT b.username, date_trunc('month', b.expense_date)::date AS event_date, SUM(price)::float AS expenses_sum, s.salary, 0.0::float AS owes
		FROM budget_schema.expense AS b
        JOIN budget_schema.salary AS s ON b.username = s.username
		AND date_trunc('month', s.store_date) = date_trunc('month', b.expense_date)
	WHERE b.expense_date BETWEEN date_trunc('month', sqlc.arg('start_time')::date)::date
		AND date_trunc('month', sqlc.arg('end_time')::date)::date + interval '1 month - 1 day'
		OR s.store_date BETWEEN date_trunc('month', sqlc.arg('start_time')::date)::date
		AND date_trunc('month', sqlc.arg('end_time')::date)::date + interval '1 month - 1 day'
	GROUP BY b.username, date_trunc('month', b.expense_date), s.salary
	ORDER BY b.username, date_trunc('month', b.expense_date), expenses_sum;

-- For some reason sqlc fails to generate go files if `date_trunc('month', b.expense_date)::date AS event_date`
-- is used later on by the GROUP BY and ORDER BY. Works fine with the psql though. Hence, a bug?
