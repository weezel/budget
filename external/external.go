package external

import "time"

type SpendingHistory struct {
	Username  string
	MonthYear time.Time
	Spending  float64
}
