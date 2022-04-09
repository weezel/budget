package external

import (
	"time"
)

type SpendingHistory struct {
	_         struct{}
	MonthYear time.Time
	Username  string
	EventName string
	Spending  float64
	Salary    float64
	ID        int64
}

type SpendingHTMLOutput struct {
	From      time.Time
	To        time.Time
	Spendings []SpendingHistory
}
