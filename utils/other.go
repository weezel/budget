package utils

import (
	"regexp"
	"strings"
)

var categoryPattern *regexp.Regexp = regexp.MustCompile(`^#[a-zA-Z1-9_-]+$`)

func GetCategory(tokens []string) string {
	for _, token := range tokens {
		if categoryPattern.MatchString(token) {
			tmp := categoryPattern.FindString(token)
			return strings.ReplaceAll(tmp, "#", "")
		}
	}

	return ""
}
