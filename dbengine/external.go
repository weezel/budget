package dbengine

import (
	"time"
	"weezel/budget/db"
)

type ExpensesVars struct {
	From      time.Time
	To        time.Time
	Spendings []*db.GetExpensesByTimespanRow
}

type StatisticsVars struct {
	From      time.Time
	To        time.Time
	Spendings []*db.StatisticsByTimespanRow
}
