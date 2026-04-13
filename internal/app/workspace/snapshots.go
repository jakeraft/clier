package workspace

import "encoding/json"

func decodeSnapshot[T any](snapshot json.RawMessage) (*T, error) {
	var s T
	return &s, json.Unmarshal(snapshot, &s)
}
