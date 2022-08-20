package utils

import "time"

// GetDate parses date in "day-month-year" format from
// the tokens and if not found, returns zero time.
func GetDate(tokens []string, format string) time.Time {
	// TODO handle time.Time{} references
	for _, token := range tokens {
		parsedTime, err := time.Parse(format, token)
		if nil != err {
			continue
		}
		return parsedTime.UTC()
	}

	return time.Time{}
}
