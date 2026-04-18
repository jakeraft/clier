package app

import (
	"strconv"
	"strings"

	remoteapi "github.com/jakeraft/clier/internal/adapter/api"
)

func itoa(v int) string { return strconv.Itoa(v) }

func joinCauses(causes []remoteapi.StatusCause) string {
	parts := make([]string, 0, len(causes))
	for _, c := range causes {
		switch {
		case c.Field != "" && c.Message != "":
			parts = append(parts, c.Field+": "+c.Message)
		case c.Message != "":
			parts = append(parts, c.Message)
		}
	}
	return strings.Join(parts, "; ")
}
