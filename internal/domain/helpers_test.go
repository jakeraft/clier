package domain

import "strings"

func containsString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

func hasPrefix(s, prefix string) bool {
	return strings.HasPrefix(s, prefix)
}
