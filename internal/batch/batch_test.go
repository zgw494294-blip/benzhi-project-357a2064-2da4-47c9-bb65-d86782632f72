package batch

import (
	"encoding/json"
	"testing"
)

func TestNewPreservesAbsentAndExplicitlyEmptyNotes(t *testing.T) {
	ab, err := New("absent", "water", []string{"s1"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	empty := ""
	eb, err := New("empty", "water", []string{"s1"}, &empty)
	if err != nil {
		t.Fatal(err)
	}
	abJSON, err := json.Marshal(ab)
	if err != nil {
		t.Fatal(err)
	}
	ebJSON, err := json.Marshal(eb)
	if err != nil {
		t.Fatal(err)
	}
	if string(abJSON) == string(ebJSON) {
		t.Fatalf("absent and empty notes should serialize differently: %s", abJSON)
	}
	if string(abJSON) != `{"id":"absent","ink_family":"water","state":"active","screens":[{"id":"s1"}]}` {
		t.Fatalf("unexpected absent-note JSON: %s", abJSON)
	}
	if string(ebJSON) != `{"id":"empty","ink_family":"water","note":"","state":"active","screens":[{"id":"s1"}]}` {
		t.Fatalf("unexpected empty-note JSON: %s", ebJSON)
	}
}

func TestNewRejectsEmptyAndDuplicateManifestEntries(t *testing.T) {
	for _, screenIDs := range [][]string{{}, {"s1", "s1"}, {"s1", ""}} {
		if _, err := New("batch", "water", screenIDs, nil); err == nil {
			t.Fatalf("expected invalid manifest for %#v", screenIDs)
		}
	}
}

func TestRecordRejectsUnknownAndDuplicateWithoutChangingOrder(t *testing.T) {
	b, err := New("b1", "plastisol", []string{"first", "second", "third"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := b.Record("second", Rework); err != nil {
		t.Fatal(err)
	}
	if err := b.Record("missing", Clean); err == nil {
		t.Fatal("expected unknown screen error")
	}
	if err := b.Record("second", Clean); err == nil {
		t.Fatal("expected duplicate screen error")
	}
	if b.Screens[0].Outcome != nil || b.Screens[2].Outcome != nil {
		t.Fatalf("neighboring screens changed: %#v", b.Screens)
	}
	if b.Screens[1].Outcome == nil || *b.Screens[1].Outcome != Rework {
		t.Fatalf("recorded outcome lost: %#v", b.Screens)
	}
}

func TestCloseDerivesImmutableAttentionReport(t *testing.T) {
	b, err := New("b1", "water", []string{"a", "b", "c"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := b.Close(); err == nil {
		t.Fatal("expected incomplete close error")
	}
	for _, update := range []struct {
		id          string
		disposition Disposition
	}{{"a", Clean}, {"b", Rework}, {"c", Retire}} {
		if err := b.Record(update.id, update.disposition); err != nil {
			t.Fatal(err)
		}
	}
	if err := b.Close(); err != nil {
		t.Fatal(err)
	}
	if b.State != Closed || b.Report == nil {
		t.Fatalf("batch did not close: %#v", b)
	}
	if *b.Screens[0].Outcome != Clean || *b.Screens[1].Outcome != Rework || *b.Screens[2].Outcome != Retire {
		t.Fatalf("manifest order changed: %#v", b.Screens)
	}
	if *b.Report != (Report{Clean: 1, Rework: 1, Retire: 1, Result: Attention}) {
		t.Fatalf("unexpected report: %#v", b.Report)
	}
	if err := b.Record("a", Clean); err == nil {
		t.Fatal("expected record-after-close error")
	}
	if err := b.Close(); err == nil {
		t.Fatal("expected repeated close error")
	}
}

func TestInspectSortsAndFiltersBatches(t *testing.T) {
	attention, err := New("z-closed", "water", []string{"a"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := attention.Record("a", Retire); err != nil {
		t.Fatal(err)
	}
	if err := attention.Close(); err != nil {
		t.Fatal(err)
	}
	allReusable, err := New("a-closed", "water", []string{"a"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := allReusable.Record("a", Clean); err != nil {
		t.Fatal(err)
	}
	if err := allReusable.Close(); err != nil {
		t.Fatal(err)
	}
	activeB, err := New("b-active", "water", []string{"a"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	activeA, err := New("a-active", "water", []string{"a"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	inspection := Inspect([]Batch{attention, activeB, allReusable, activeA})
	if len(inspection.Active) != 2 || inspection.Active[0].ID != "a-active" || inspection.Active[1].ID != "b-active" {
		t.Fatalf("active batches not deterministic: %#v", inspection.Active)
	}
	if len(inspection.ClosedRequiringAttention) != 1 || inspection.ClosedRequiringAttention[0].ID != "z-closed" {
		t.Fatalf("attention batches not filtered: %#v", inspection.ClosedRequiringAttention)
	}
}
