package ledger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"meshturn/internal/batch"
)

func TestSaveAndLoadPreserveLedgerAndOptionalNotes(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "ledger.json")
	result := New()
	absent, err := batch.New("absent", "water", []string{"s1"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	empty := ""
	emptyNote, err := batch.New("empty", "water", []string{"s1"}, &empty)
	if err != nil {
		t.Fatal(err)
	}
	if err := result.Add(absent); err != nil {
		t.Fatal(err)
	}
	if err := result.Add(emptyNote); err != nil {
		t.Fatal(err)
	}
	if err := result.Save(path); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Batches) != 2 || loaded.Batches[0].Note != nil || loaded.Batches[1].Note == nil || *loaded.Batches[1].Note != "" {
		t.Fatalf("optional note semantics not preserved: %#v", loaded.Batches)
	}
	if _, err := os.Stat(TemporaryPath(path)); !os.IsNotExist(err) {
		t.Fatalf("temporary file remains after successful save: %v", err)
	}
}

func TestLoadSurfacesCorruptData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ledger.json")
	if err := os.WriteFile(path, []byte(`{"version":1,"batches":[`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil || !strings.Contains(err.Error(), "decode ledger") {
		t.Fatalf("expected decode error, got %v", err)
	}
}

func TestFailedReplaceRemovesSiblingTemporaryFile(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "ledger.json")
	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := New().Save(path); err == nil {
		t.Fatal("expected rename failure when ledger path is a directory")
	}
	if _, err := os.Stat(TemporaryPath(path)); !os.IsNotExist(err) {
		t.Fatalf("temporary file remains after failed replace: %v", err)
	}
}

func TestLoadRejectsUnsupportedVersionAndTrailingValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ledger.json")
	for _, contents := range []string{
		`{"version":2,"batches":[]}`,
		`{"version":1,"batches":[]} {"version":1,"batches":[]}`,
	} {
		if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(path); err == nil {
			t.Fatalf("expected invalid ledger for %s", contents)
		}
	}
}
