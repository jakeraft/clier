package workspace

import "encoding/json"

func decodeSnapshot[T any](snapshot json.RawMessage) (*T, error) {
	var s T
	return &s, json.Unmarshal(snapshot, &s)
}

func decodeSnapshotInto(snapshot json.RawMessage, target any) error {
	return json.Unmarshal(snapshot, target)
}
