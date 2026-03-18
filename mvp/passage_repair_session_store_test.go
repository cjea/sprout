package mvp

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestJSONPassageRepairSessionStoreSavesAndLoadsSession(t *testing.T) {
	store := JSONPassageRepairSessionStore{Path: filepath.Join(t.TempDir(), "repair-session.json")}
	snapshot := newRepairSnapshotFixture(t)
	session, err := NewPassageRepairSession(snapshot)
	if err != nil {
		t.Fatalf("new session: %v", err)
	}

	if err := store.SavePassageRepairSession(session); err != nil {
		t.Fatalf("save session: %v", err)
	}
	loaded, err := store.LoadPassageRepairSession()
	if err != nil {
		t.Fatalf("load session: %v", err)
	}
	if loaded.Current.Revision != 0 {
		t.Fatalf("got revision %d want 0", loaded.Current.Revision)
	}
}

func TestJSONPassageRepairSessionStoreRejectsMissingFile(t *testing.T) {
	store := JSONPassageRepairSessionStore{Path: filepath.Join(t.TempDir(), "missing.json")}
	_, err := store.LoadPassageRepairSession()
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("got err %v want %v", err, ErrNotFound)
	}
}
