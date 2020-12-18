package external

import "time"

type SpendingHistory struct {
	_         struct{}
	Username  string
	MonthYear time.Time
	Spending  float64
	EventName string
}
