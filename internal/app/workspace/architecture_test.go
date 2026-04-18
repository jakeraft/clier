package workspace

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestServiceUsesRemoteClientPort(t *testing.T) {
	t.Parallel()

	field, ok := reflect.TypeFor[Service]().FieldByName("client")
	if !ok {
		t.Fatal("Service.client field missing")
	}
	if field.Type.Kind() != reflect.Interface {
		t.Fatalf("Service.client should be an interface port, got %s", field.Type)
	}
	if field.Type.Name() != "RemoteWorkspaceClient" {
		t.Fatalf("Service.client should use RemoteWorkspaceClient, got %s", field.Type.Name())
	}
}

func TestWorkspaceServiceDoesNotDecodeResolveWireFormat(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile(filepath.Join(".", "service.go"))
	if err != nil {
		t.Fatalf("read service.go: %v", err)
	}
	source := string(data)
	for _, forbidden := range []string{"snapshotRef", "decodeSnapshot", "decodeSnapshotInto"} {
		if strings.Contains(source, forbidden) {
			t.Fatalf("service.go should not contain resolve wire-format decoder %q", forbidden)
		}
	}
}
