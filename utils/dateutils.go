package utils

import "time"

// GetDate parses date in "day-month-year" format from
// the tokens and if not found, returns current date
func GetDate(tokens []string, format string) time.Time {
	for _, token := range tokens {
		parsedTime, err := time.Parse(format, token)
		if nil != err {
			continue
		}
		return parsedTime.UTC()
	}

	now := time.Now().UTC()
	return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
}
