package mvp

import (
	"errors"
	"testing"
)

func TestMemoryPassageRepairStoreSavesAndLoadsSnapshots(t *testing.T) {
	store := NewMemoryPassageRepairStore()
	snapshot := newRepairSnapshotFixture(t)

	saved, err := store.SavePassageRepairSnapshot(snapshot)
	if err != nil {
		t.Fatalf("save snapshot: %v", err)
	}
	if saved.Revision != 0 {
		t.Fatalf("got revision %d want 0", saved.Revision)
	}

	loaded, err := store.LoadPassageRepairSnapshot(snapshot.SessionID, snapshot.Revision)
	if err != nil {
		t.Fatalf("load snapshot: %v", err)
	}
	if len(loaded.Passages) != len(snapshot.Passages) {
		t.Fatalf("got %d passages want %d", len(loaded.Passages), len(snapshot.Passages))
	}
}

func TestMemoryPassageRepairStoreLoadsLatestSnapshot(t *testing.T) {
	store := NewMemoryPassageRepairStore()
	snapshot := newRepairSnapshotFixture(t)
	if _, err := store.SavePassageRepairSnapshot(snapshot); err != nil {
		t.Fatalf("save first snapshot: %v", err)
	}

	target, _ := NewPassageRepairTarget(snapshot.OpinionID, []PassageID{snapshot.Passages[0].PassageID})
	operation, _ := NewAdminPassageOperation(AdminPassageOperationMergeWithNext, target, nil)
	next, err := ApplyAdminPassageOperation(snapshot, operation)
	if err != nil {
		t.Fatalf("apply operation: %v", err)
	}
	if _, err := store.SavePassageRepairSnapshot(next); err != nil {
		t.Fatalf("save second snapshot: %v", err)
	}

	latest, err := store.LoadLatestPassageRepairSnapshot(snapshot.SessionID)
	if err != nil {
		t.Fatalf("load latest snapshot: %v", err)
	}
	if latest.Revision != 1 {
		t.Fatalf("got revision %d want 1", latest.Revision)
	}
}

func TestMemoryPassageRepairStoreRejectsMissingSnapshots(t *testing.T) {
	store := NewMemoryPassageRepairStore()
	_, err := store.LoadPassageRepairSnapshot("missing", 0)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("got err %v want %v", err, ErrNotFound)
	}
	_, err = store.LoadLatestPassageRepairSnapshot("missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("got err %v want %v", err, ErrNotFound)
	}
}
