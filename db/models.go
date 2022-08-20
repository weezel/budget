// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.14.0

package db

import (
	"time"
)

type BudgetSchemaExpense struct {
	ID          int32     `json:"id"`
	Username    string    `json:"username"`
	ShopName    string    `json:"shop_name"`
	Category    string    `json:"category"`
	Price       float64   `json:"price"`
	ExpenseDate time.Time `json:"expense_date"`
}

type BudgetSchemaSalary struct {
	ID        int32     `json:"id"`
	Username  string    `json:"username"`
	Salary    float64   `json:"salary"`
	StoreDate time.Time `json:"store_date"`
}
