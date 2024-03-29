// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.14.0

package db

import (
	"context"
)

type Querier interface {
	//
	// Expenses
	//
	AddExpense(ctx context.Context, arg AddExpenseParams) (int32, error)
	//
	// Salaries
	//
	AddSalary(ctx context.Context, arg AddSalaryParams) (int32, error)
	DeleteExpenseByID(ctx context.Context, arg DeleteExpenseByIDParams) (*BudgetSchemaExpense, error)
	DeleteSalaryByID(ctx context.Context, arg DeleteSalaryByIDParams) (*BudgetSchemaSalary, error)
	GetAggrExpensesByTimespan(ctx context.Context, arg GetAggrExpensesByTimespanParams) ([]*GetAggrExpensesByTimespanRow, error)
	GetExpensesByTimespan(ctx context.Context, arg GetExpensesByTimespanParams) ([]*GetExpensesByTimespanRow, error)
	GetSalariesByTimespan(ctx context.Context, arg GetSalariesByTimespanParams) ([]*GetSalariesByTimespanRow, error)
	GetUserSalaryByMonth(ctx context.Context, arg GetUserSalaryByMonthParams) (float64, error)
	//
	// Miscellaneous
	//
	StatisticsAggrByTimespan(ctx context.Context, arg StatisticsAggrByTimespanParams) ([]*StatisticsAggrByTimespanRow, error)
}

var _ Querier = (*Queries)(nil)
