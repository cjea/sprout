package mvp

import (
	"context"
	"testing"
	"time"
)

func TestSQLiteRepairAuditRoundTrip(t *testing.T) {
	storage, cleanup := newSQLiteFixtureStorage(t)
	defer cleanup()

	opinion := sampleOpinionFixture(t)
	if _, err := saveOpinion(storage, opinion); err != nil {
		t.Fatalf("save opinion: %v", err)
	}
	passages := sampleRepairAuditPassagesFixture(t, opinion.OpinionID, opinion.Sections[0].SectionID)
	if _, err := savePassages(storage, passages); err != nil {
		t.Fatalf("save passages: %v", err)
	}

	session, err := LoadOrStartAuditedPassageRepairSession(context.Background(), storage, opinion.OpinionID, "admin", "browser", passages)
	if err != nil {
		t.Fatalf("start session: %v", err)
	}
	target, _ := NewPassageRepairTarget(opinion.OpinionID, []PassageID{passages[0].PassageID})
	operation, _ := NewAdminPassageOperation(AdminPassageOperationMergeWithNext, target, nil)
	before := session.Current
	if err := session.Apply(operation); err != nil {
		t.Fatalf("apply operation: %v", err)
	}
	if err := RecordAuditedPassageRepairOperation(context.Background(), storage, "admin", "browser", operation, before, session.Current, time.Date(2026, time.March, 17, 1, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("record operation: %v", err)
	}

	entries, err := storage.ListPassageRepairAudit(context.Background(), opinion.OpinionID, "admin")
	if err != nil {
		t.Fatalf("list audit: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries want 1", len(entries))
	}
	if entries[0].OperationKind != AdminPassageOperationMergeWithNext {
		t.Fatalf("got operation %q", entries[0].OperationKind)
	}

	loaded, err := storage.LoadLatestPassageRepairSession(context.Background(), opinion.OpinionID, "admin")
	if err != nil {
		t.Fatalf("load latest session: %v", err)
	}
	if loaded.Current.Revision != 1 {
		t.Fatalf("got revision %d want 1", loaded.Current.Revision)
	}
}

func TestLoadOrStartAuditedPassageRepairSessionReconcilesStaleSnapshot(t *testing.T) {
	storage, cleanup := newSQLiteFixtureStorage(t)
	defer cleanup()

	opinion := sampleOpinionFixture(t)
	if _, err := saveOpinion(storage, opinion); err != nil {
		t.Fatalf("save opinion: %v", err)
	}

	oldPassages := sampleRepairAuditPassagesFixture(t, opinion.OpinionID, opinion.Sections[0].SectionID)
	session, err := LoadOrStartAuditedPassageRepairSession(context.Background(), storage, opinion.OpinionID, "admin", "browser", oldPassages)
	if err != nil {
		t.Fatalf("start session: %v", err)
	}
	target, _ := NewPassageRepairTarget(opinion.OpinionID, []PassageID{oldPassages[0].PassageID})
	operation, _ := NewAdminPassageOperation(AdminPassageOperationMergeWithNext, target, nil)
	before := session.Current
	if err := session.Apply(operation); err != nil {
		t.Fatalf("apply operation: %v", err)
	}
	if err := RecordAuditedPassageRepairOperation(context.Background(), storage, "admin", "browser", operation, before, session.Current, time.Date(2026, time.March, 17, 1, 5, 0, 0, time.UTC)); err != nil {
		t.Fatalf("record operation: %v", err)
	}

	currentPassageID, _ := NewPassageID("current-1")
	currentPassage, err := buildPassageFromSourceText(currentPassageID, opinion.OpinionID, opinion.Sections[0].SectionID, 0, 0, 1, 1, "Current repaired passage.", true)
	if err != nil {
		t.Fatalf("build current passage: %v", err)
	}
	reconciled, err := LoadOrStartAuditedPassageRepairSession(context.Background(), storage, opinion.OpinionID, "admin", "browser", []Passage{currentPassage})
	if err != nil {
		t.Fatalf("reconcile session: %v", err)
	}
	if len(reconciled.Current.Passages) != 1 || reconciled.Current.Passages[0].PassageID != currentPassageID {
		t.Fatalf("expected reconciled current passage set, got %#v", reconciled.Current.Passages)
	}
	if reconciled.Current.Revision != session.Current.Revision {
		t.Fatalf("got revision %d want %d", reconciled.Current.Revision, session.Current.Revision)
	}
}

func TestSQLiteRepairAuditUndoGetsNewAuditRevision(t *testing.T) {
	storage, cleanup := newSQLiteFixtureStorage(t)
	defer cleanup()

	opinion := sampleOpinionFixture(t)
	if _, err := saveOpinion(storage, opinion); err != nil {
		t.Fatalf("save opinion: %v", err)
	}
	passages := sampleRepairAuditPassagesFixture(t, opinion.OpinionID, opinion.Sections[0].SectionID)
	if _, err := savePassages(storage, passages); err != nil {
		t.Fatalf("save passages: %v", err)
	}

	session, err := LoadOrStartAuditedPassageRepairSession(context.Background(), storage, opinion.OpinionID, "admin", "browser", passages)
	if err != nil {
		t.Fatalf("start session: %v", err)
	}
	target, _ := NewPassageRepairTarget(opinion.OpinionID, []PassageID{passages[0].PassageID})
	applyOperation, _ := NewAdminPassageOperation(AdminPassageOperationMergeWithNext, target, nil)
	beforeApply := session.Current
	if err := session.Apply(applyOperation); err != nil {
		t.Fatalf("apply operation: %v", err)
	}
	if err := RecordAuditedPassageRepairOperation(context.Background(), storage, "admin", "browser", applyOperation, beforeApply, session.Current, time.Date(2026, time.March, 17, 1, 10, 0, 0, time.UTC)); err != nil {
		t.Fatalf("record apply: %v", err)
	}

	beforeUndo := session.Current
	if err := session.Undo(); err != nil {
		t.Fatalf("undo session: %v", err)
	}
	undoTarget, _ := NewPassageRepairTarget(opinion.OpinionID, []PassageID{session.Current.Passages[0].PassageID})
	undoOperation, _ := NewAdminPassageOperation(AdminPassageOperationUndo, undoTarget, nil)
	if err := RecordAuditedPassageRepairOperation(context.Background(), storage, "admin", "browser", undoOperation, beforeUndo, session.Current, time.Date(2026, time.March, 17, 1, 11, 0, 0, time.UTC)); err != nil {
		t.Fatalf("record undo: %v", err)
	}

	entries, err := storage.ListPassageRepairAudit(context.Background(), opinion.OpinionID, "admin")
	if err != nil {
		t.Fatalf("list audit: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries want 2", len(entries))
	}
	if entries[0].Revision != 1 || entries[1].Revision != 2 {
		t.Fatalf("got revisions %d and %d, want 1 and 2", entries[0].Revision, entries[1].Revision)
	}
	if entries[1].After.Revision != 0 {
		t.Fatalf("got undone snapshot revision %d, want 0", entries[1].After.Revision)
	}
}

func sampleRepairAuditPassagesFixture(t *testing.T, opinionID OpinionID, sectionID SectionID) []Passage {
	t.Helper()
	firstID, _ := NewPassageID("repair-a-1")
	secondID, _ := NewPassageID("repair-a-2")
	first, err := buildPassageFromSourceText(firstID, opinionID, sectionID, 0, 0, 1, 1, "The Board denied relief.", true)
	if err != nil {
		t.Fatalf("first passage: %v", err)
	}
	second, err := buildPassageFromSourceText(secondID, opinionID, sectionID, 1, 1, 1, 1, "The court of appeals affirmed.", true)
	if err != nil {
		t.Fatalf("second passage: %v", err)
	}
	return []Passage{first, second}
}
