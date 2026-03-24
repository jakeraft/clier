package cmd

import (
	"encoding/json"
	"os"
)

func printJSON(v any) error {
	return json.NewEncoder(os.Stdout).Encode(v)
}
