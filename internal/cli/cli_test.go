package cli

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunCompletesWorkflowAcrossCLIInvocations(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ledger.json")
	var output bytes.Buffer
	commands := [][]string{
		{"--ledger", path, "open", "--id", "batch-2", "--ink-family", "water", "--screens", "s1,s2", "--note", ""},
		{"--ledger", path, "record", "--id", "batch-2", "--screen", "s1", "--outcome", "clean"},
		{"--ledger", path, "record", "--id", "batch-2", "--screen", "s2", "--outcome", "rework"},
		{"--ledger", path, "close", "--id", "batch-2"},
		{"--ledger", path, "show", "--id", "batch-2"},
	}
	for _, args := range commands {
		output.Reset()
		if err := Run(args, &output, &output); err != nil {
			t.Fatalf("%v: %v", strings.Join(args, " "), err)
		}
		var value map[string]any
		if err := json.Unmarshal(output.Bytes(), &value); err != nil {
			t.Fatalf("invalid JSON for %v: %v", args, err)
		}
	}
	if !strings.Contains(output.String(), `"note":""`) || !strings.Contains(output.String(), `"result":"attention"`) {
		t.Fatalf("show output lost report or explicit empty note: %s", output.String())
	}

	output.Reset()
	if err := Run([]string{"--ledger", path, "attention"}, &output, &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), `"closed_requiring_attention"`) || !strings.Contains(output.String(), `"batch-2"`) {
		t.Fatalf("attention output missing closed batch: %s", output.String())
	}
}

func TestRunRejectsInvalidRecordWithoutSavingChanges(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ledger.json")
	var output bytes.Buffer
	if err := Run([]string{"--ledger", path, "open", "--id", "batch-1", "--ink-family", "water", "--screens", "s1,s2"}, &output, &output); err != nil {
		t.Fatal(err)
	}
	if err := Run([]string{"--ledger", path, "record", "--id", "batch-1", "--screen", "unknown", "--outcome", "clean"}, &output, &output); err == nil {
		t.Fatal("expected unknown-screen error")
	}
	output.Reset()
	if err := Run([]string{"--ledger", path, "show", "--id", "batch-1"}, &output, &output); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output.String(), `"outcome"`) {
		t.Fatalf("invalid record changed ledger: %s", output.String())
	}
}
