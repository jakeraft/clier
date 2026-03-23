package db

import "testing"

func TestNewStore(t *testing.T) {
	t.Run("InMemory_InitializesSchemaSuccessfully", func(t *testing.T) {
		store, err := NewStore(":memory:")
		if err != nil {
			t.Fatalf("NewStore: %v", err)
		}
		defer store.Close()

		if store.db == nil {
			t.Fatal("expected non-nil db")
		}
		if store.queries == nil {
			t.Fatal("expected non-nil queries")
		}
	})

	t.Run("InMemory_ForeignKeysEnabled", func(t *testing.T) {
		store, err := NewStore(":memory:")
		if err != nil {
			t.Fatalf("NewStore: %v", err)
		}
		defer store.Close()

		var fkEnabled int
		err = store.db.QueryRow("PRAGMA foreign_keys").Scan(&fkEnabled)
		if err != nil {
			t.Fatalf("query pragma: %v", err)
		}
		if fkEnabled != 1 {
			t.Errorf("foreign_keys = %d, want 1", fkEnabled)
		}
	})
}
