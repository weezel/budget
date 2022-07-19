// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.14.0

package db

import (
	"context"
	"time"

	"github.com/jackc/pgtype"
)

type Querier interface {
	AddExpense(ctx context.Context, arg AddExpenseParams) error
	AddSalary(ctx context.Context, arg AddSalaryParams) error
	GetExpenses(ctx context.Context, dateTrunc interface{}) ([]GetExpensesRow, error)
	GetMonthlyExpenses(ctx context.Context, expenseDate time.Time) ([]GetMonthlyExpensesRow, error)
	GetMonthlySalaries(ctx context.Context, arg GetMonthlySalariesParams) ([]GetMonthlySalariesRow, error)
	GetSalaryByMonth(ctx context.Context, arg GetSalaryByMonthParams) (pgtype.Numeric, error)
	GetSpendingTimespan(ctx context.Context, arg GetSpendingTimespanParams) ([]GetSpendingTimespanRow, error)
}

var _ Querier = (*Queries)(nil)
