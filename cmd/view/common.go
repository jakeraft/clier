package view

// Items wraps list-shaped responses so every command can keep the
// top-level stdout contract as a JSON object.
type Items[T any] struct {
	Items []T `json:"items"`
}

type DeletedResult struct {
	Deleted string `json:"deleted"`
}

// ItemsOf normalizes nil slices to [] for stable agent-facing output.
func ItemsOf[T any](items []T) Items[T] {
	if items == nil {
		items = []T{}
	}
	return Items[T]{Items: items}
}

func DeletedOf(name string) DeletedResult {
	return DeletedResult{Deleted: name}
}
