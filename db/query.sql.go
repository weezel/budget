// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.14.0
// source: query.sql

package db

import (
	"context"
	"time"

	"github.com/jackc/pgtype"
)

const addExpense = `-- name: AddExpense :exec
INSERT INTO budget(
	username,
	shop_name,
	category,
	price,
	expense_date
) VALUES (
	$1, $2, $3, $4, $5
)
`

type AddExpenseParams struct {
	Username    string         `json:"username"`
	ShopName    string         `json:"shop_name"`
	Category    string         `json:"category"`
	Price       pgtype.Numeric `json:"price"`
	ExpenseDate time.Time      `json:"expense_date"`
}

func (q *Queries) AddExpense(ctx context.Context, arg AddExpenseParams) error {
	_, err := q.db.Exec(ctx, addExpense,
		arg.Username,
		arg.ShopName,
		arg.Category,
		arg.Price,
		arg.ExpenseDate,
	)
	return err
}

const addSalary = `-- name: AddSalary :exec
INSERT INTO salary(
	username,
	salary,
	store_date
) VALUES (
	$1, $2, $3
)
`

type AddSalaryParams struct {
	Username  string         `json:"username"`
	Salary    pgtype.Numeric `json:"salary"`
	StoreDate time.Time      `json:"store_date"`
}

func (q *Queries) AddSalary(ctx context.Context, arg AddSalaryParams) error {
	_, err := q.db.Exec(ctx, addSalary, arg.Username, arg.Salary, arg.StoreDate)
	return err
}

const getExpenses = `-- name: GetExpenses :many
SELECT username, expense_date, SUM(price)::float AS expenses_sum FROM budget
	GROUP BY expense_date, username
	HAVING expense_date = date_trunc('month', $1)
	ORDER BY username
`

type GetExpensesRow struct {
	Username    string    `json:"username"`
	ExpenseDate time.Time `json:"expense_date"`
	ExpensesSum float64   `json:"expenses_sum"`
}

func (q *Queries) GetExpenses(ctx context.Context, dateTrunc interface{}) ([]GetExpensesRow, error) {
	rows, err := q.db.Query(ctx, getExpenses, dateTrunc)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetExpensesRow
	for rows.Next() {
		var i GetExpensesRow
		if err := rows.Scan(&i.Username, &i.ExpenseDate, &i.ExpensesSum); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getMonthlyExpenses = `-- name: GetMonthlyExpenses :many
SELECT id, username, expense_date, shop_name, price FROM budget
	GROUP BY username, expense_date, shop_name, price
	HAVING expense_date = $1
	ORDER BY username, expense_date, shop_name, price
`

type GetMonthlyExpensesRow struct {
	ID          int32          `json:"id"`
	Username    string         `json:"username"`
	ExpenseDate time.Time      `json:"expense_date"`
	ShopName    string         `json:"shop_name"`
	Price       pgtype.Numeric `json:"price"`
}

func (q *Queries) GetMonthlyExpenses(ctx context.Context, expenseDate time.Time) ([]GetMonthlyExpensesRow, error) {
	rows, err := q.db.Query(ctx, getMonthlyExpenses, expenseDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetMonthlyExpensesRow
	for rows.Next() {
		var i GetMonthlyExpensesRow
		if err := rows.Scan(
			&i.ID,
			&i.Username,
			&i.ExpenseDate,
			&i.ShopName,
			&i.Price,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getMonthlySalaries = `-- name: GetMonthlySalaries :many
SELECT username, salary, store_date FROM salary
	WHERE store_date BETWEEN date_trunc('month', $1::date)::date
		AND (date_trunc('month', $2::date) + interval '1 month - 1 day')::date
	GROUP BY username, store_date, salary
	ORDER BY username, store_date
`

type GetMonthlySalariesParams struct {
	StartMonth time.Time `json:"StartMonth"`
	EndMonth   time.Time `json:"EndMonth"`
}

type GetMonthlySalariesRow struct {
	Username  string         `json:"username"`
	Salary    pgtype.Numeric `json:"salary"`
	StoreDate time.Time      `json:"store_date"`
}

func (q *Queries) GetMonthlySalaries(ctx context.Context, arg GetMonthlySalariesParams) ([]GetMonthlySalariesRow, error) {
	rows, err := q.db.Query(ctx, getMonthlySalaries, arg.StartMonth, arg.EndMonth)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetMonthlySalariesRow
	for rows.Next() {
		var i GetMonthlySalariesRow
		if err := rows.Scan(&i.Username, &i.Salary, &i.StoreDate); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getSalaryByMonth = `-- name: GetSalaryByMonth :one
SELECT salary FROM salary
	WHERE username = $1
		AND store_date = $2
	LIMIT 1
`

type GetSalaryByMonthParams struct {
	Username  string    `json:"username"`
	StoreDate time.Time `json:"store_date"`
}

func (q *Queries) GetSalaryByMonth(ctx context.Context, arg GetSalaryByMonthParams) (pgtype.Numeric, error) {
	row := q.db.QueryRow(ctx, getSalaryByMonth, arg.Username, arg.StoreDate)
	var salary pgtype.Numeric
	err := row.Scan(&salary)
	return salary, err
}

const getSpendingTimespan = `-- name: GetSpendingTimespan :many
SELECT b.username, b.expense_date AS expense_date, SUM(price)::float AS expenses_sum, s.salary
		FROM budget AS b
        INNER JOIN salary AS s ON b.username = s.username
		AND s.store_date = b.expense_date
	WHERE b.expense_date BETWEEN date_trunc('month', $1::date)::date
		AND (date_trunc('month', $2::date) + interval '1 month - 1 day')::date
		AND s.store_date != NULL
		AND b.expense_date != NULL
	GROUP BY b.username, b.expense_date
	ORDER BY b.username, b.expense_date, expenses
`

type GetSpendingTimespanParams struct {
	StartMonth time.Time `json:"StartMonth"`
	EndMonth   time.Time `json:"EndMonth"`
}

type GetSpendingTimespanRow struct {
	Username    string         `json:"username"`
	ExpenseDate time.Time      `json:"expense_date"`
	ExpensesSum float64        `json:"expenses_sum"`
	Salary      pgtype.Numeric `json:"salary"`
}

func (q *Queries) GetSpendingTimespan(ctx context.Context, arg GetSpendingTimespanParams) ([]GetSpendingTimespanRow, error) {
	rows, err := q.db.Query(ctx, getSpendingTimespan, arg.StartMonth, arg.EndMonth)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetSpendingTimespanRow
	for rows.Next() {
		var i GetSpendingTimespanRow
		if err := rows.Scan(
			&i.Username,
			&i.ExpenseDate,
			&i.ExpensesSum,
			&i.Salary,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
