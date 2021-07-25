package external

import (
	"time"
)

type SpendingHistory struct {
	_         struct{}
	ID        int64
	Username  string
	MonthYear time.Time
	Spending  float64
	Salary    float64
	EventName string
}

type SpendingHTMLOutput struct {
	From      time.Time
	To        time.Time
	Spendings map[time.Time][]SpendingHistory
}

// Helper function for HTML template rendering
func (s SpendingHTMLOutput) GetOne() SpendingHistory {
	for _, val := range s.Spendings {
		return val[0]
	}
	return SpendingHistory{}
}
