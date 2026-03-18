package mvp

import (
	"errors"
	"strings"
	"testing"
)

func TestApplyAdminPassageOperationMergeWithNext(t *testing.T) {
	snapshot := newRepairSnapshotFixture(t)
	passageID := snapshot.Passages[0].PassageID
	target, _ := NewPassageRepairTarget(snapshot.OpinionID, []PassageID{passageID})
	operation, err := NewAdminPassageOperation(AdminPassageOperationMergeWithNext, target, nil)
	if err != nil {
		t.Fatalf("new operation: %v", err)
	}

	next, err := ApplyAdminPassageOperation(snapshot, operation)
	if err != nil {
		t.Fatalf("apply operation: %v", err)
	}
	if len(next.Passages) != 2 {
		t.Fatalf("got %d passages want 2", len(next.Passages))
	}
	if next.Passages[0].SentenceStart != 0 || next.Passages[0].SentenceEnd != 3 {
		t.Fatalf("unexpected sentence range: %+v", next.Passages[0])
	}
}

func TestApplyAdminPassageOperationSplitAtSentence(t *testing.T) {
	snapshot := newRepairSnapshotFixture(t)
	passageID := snapshot.Passages[0].PassageID
	target, _ := NewPassageRepairTarget(snapshot.OpinionID, []PassageID{passageID})
	splitAfter := SentenceNo(0)
	operation, err := NewAdminPassageOperation(AdminPassageOperationSplitAtSentence, target, &splitAfter)
	if err != nil {
		t.Fatalf("new operation: %v", err)
	}

	next, err := ApplyAdminPassageOperation(snapshot, operation)
	if err != nil {
		t.Fatalf("apply operation: %v", err)
	}
	if len(next.Passages) != 4 {
		t.Fatalf("got %d passages want 4", len(next.Passages))
	}
	if next.Passages[0].SentenceStart != 0 || next.Passages[0].SentenceEnd != 0 {
		t.Fatalf("unexpected left passage: %+v", next.Passages[0])
	}
	if next.Passages[1].SentenceStart != 1 || next.Passages[1].SentenceEnd != 1 {
		t.Fatalf("unexpected right passage: %+v", next.Passages[1])
	}
}

func TestApplyAdminPassageOperationMovesSentencesBetweenAdjacentPassages(t *testing.T) {
	snapshot := newRepairSnapshotFixture(t)

	moveLastTarget, _ := NewPassageRepairTarget(snapshot.OpinionID, []PassageID{snapshot.Passages[0].PassageID})
	moveLast, _ := NewAdminPassageOperation(AdminPassageOperationMoveLastSentenceNext, moveLastTarget, nil)
	afterMoveLast, err := ApplyAdminPassageOperation(snapshot, moveLast)
	if err != nil {
		t.Fatalf("move last sentence: %v", err)
	}
	if afterMoveLast.Passages[0].SentenceEnd != 0 {
		t.Fatalf("expected first passage to lose last sentence: %+v", afterMoveLast.Passages[0])
	}
	if afterMoveLast.Passages[1].SentenceStart != 1 {
		t.Fatalf("expected second passage to gain moved sentence: %+v", afterMoveLast.Passages[1])
	}

	moveFirstTarget, _ := NewPassageRepairTarget(snapshot.OpinionID, []PassageID{snapshot.Passages[1].PassageID})
	moveFirst, _ := NewAdminPassageOperation(AdminPassageOperationMoveFirstSentencePrev, moveFirstTarget, nil)
	afterMoveFirst, err := ApplyAdminPassageOperation(snapshot, moveFirst)
	if err != nil {
		t.Fatalf("move first sentence: %v", err)
	}
	if afterMoveFirst.Passages[0].SentenceEnd != 2 {
		t.Fatalf("expected first passage to gain moved sentence: %+v", afterMoveFirst.Passages[0])
	}
	if afterMoveFirst.Passages[1].SentenceStart != 3 {
		t.Fatalf("expected second passage to lose first sentence: %+v", afterMoveFirst.Passages[1])
	}
}

func TestApplyAdminPassageOperationDropAndRestore(t *testing.T) {
	snapshot := newRepairSnapshotFixture(t)

	dropTarget, _ := NewPassageRepairTarget(snapshot.OpinionID, []PassageID{snapshot.Passages[1].PassageID})
	dropOperation, _ := NewAdminPassageOperation(AdminPassageOperationDropPassage, dropTarget, nil)
	dropped, err := ApplyAdminPassageOperation(snapshot, dropOperation)
	if err != nil {
		t.Fatalf("drop passage: %v", err)
	}
	if len(dropped.Passages) != 2 {
		t.Fatalf("got %d passages want 2", len(dropped.Passages))
	}
	if len(dropped.Dropped) != 1 {
		t.Fatalf("got %d dropped passages want 1", len(dropped.Dropped))
	}

	restoreTarget, _ := NewPassageRepairTarget(snapshot.OpinionID, []PassageID{snapshot.Passages[1].PassageID})
	restoreOperation, _ := NewAdminPassageOperation(AdminPassageOperationRestorePassage, restoreTarget, nil)
	restored, err := ApplyAdminPassageOperation(dropped, restoreOperation)
	if err != nil {
		t.Fatalf("restore passage: %v", err)
	}
	if len(restored.Passages) != 3 {
		t.Fatalf("got %d passages want 3", len(restored.Passages))
	}
	if len(restored.Dropped) != 0 {
		t.Fatalf("expected dropped passages to be empty")
	}
}

func TestApplyAdminPassageOperationRemoveRunningHeader(t *testing.T) {
	opinionID, _ := NewOpinionID("24-777_9ol1")
	sectionID, _ := NewSectionID("syllabus")
	passageID, _ := NewPassageID("syllabus-header")
	passage, err := buildPassageFromSourceText(
		passageID,
		opinionID,
		sectionID,
		0,
		0,
		2,
		2,
		"Held: The INA requires application of the substantial-evidence standard 2 URIAS-ORELLANA v. BONDI Syllabus to the agency's determination whether a given set of undisputed facts rises to the level of persecution under §1101(a)(42)(A). Pp. 5-13.",
		true,
	)
	if err != nil {
		t.Fatalf("build passage: %v", err)
	}
	snapshot, err := NewPassageRepairSnapshot("repair-session", 0, opinionID, []Passage{passage}, nil)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	target, _ := NewPassageRepairTarget(opinionID, []PassageID{passageID})
	operation, _ := NewAdminPassageOperation(AdminPassageOperationRemoveRunningHeader, target, nil)

	next, err := ApplyAdminPassageOperation(snapshot, operation)
	if err != nil {
		t.Fatalf("apply operation: %v", err)
	}
	if strings.Contains(string(next.Passages[0].Text), "URIAS-ORELLANA v. BONDI Syllabus") {
		t.Fatalf("expected running header to be removed: %q", next.Passages[0].Text)
	}
}

func TestPassageRepairSessionUndo(t *testing.T) {
	snapshot := newRepairSnapshotFixture(t)
	session, err := NewPassageRepairSession(snapshot)
	if err != nil {
		t.Fatalf("new session: %v", err)
	}

	target, _ := NewPassageRepairTarget(snapshot.OpinionID, []PassageID{snapshot.Passages[0].PassageID})
	operation, _ := NewAdminPassageOperation(AdminPassageOperationMergeWithNext, target, nil)
	if err := session.Apply(operation); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(session.Current.Passages) != 2 {
		t.Fatalf("got %d passages want 2", len(session.Current.Passages))
	}
	if err := session.Undo(); err != nil {
		t.Fatalf("undo: %v", err)
	}
	if len(session.Current.Passages) != 3 {
		t.Fatalf("got %d passages want 3", len(session.Current.Passages))
	}
	if len(session.History) != 0 {
		t.Fatalf("expected empty history after undo")
	}
}

func TestPassageRepairSessionUndoRequiresHistory(t *testing.T) {
	snapshot := newRepairSnapshotFixture(t)
	session, err := NewPassageRepairSession(snapshot)
	if err != nil {
		t.Fatalf("new session: %v", err)
	}
	if !errors.Is(session.Undo(), ErrPassageRepairNoHistory) {
		t.Fatalf("expected empty history error")
	}
}

func TestApplyAdminPassageOperationRejectsInvalidAdjacency(t *testing.T) {
	snapshot := newRepairSnapshotFixture(t)
	target, _ := NewPassageRepairTarget(snapshot.OpinionID, []PassageID{snapshot.Passages[len(snapshot.Passages)-1].PassageID})
	operation, _ := NewAdminPassageOperation(AdminPassageOperationMergeWithNext, target, nil)

	_, err := ApplyAdminPassageOperation(snapshot, operation)
	if !errors.Is(err, ErrPassageRepairNoAdjacentPassage) {
		t.Fatalf("got err %v want %v", err, ErrPassageRepairNoAdjacentPassage)
	}
}

func newRepairSnapshotFixture(t *testing.T) PassageRepairSnapshot {
	t.Helper()

	opinionID, _ := NewOpinionID("24-777_9ol1")
	sectionID, _ := NewSectionID("syllabus")
	firstID, _ := NewPassageID("syllabus-1")
	secondID, _ := NewPassageID("syllabus-2")
	thirdID, _ := NewPassageID("syllabus-3")

	first, err := buildPassageFromSourceText(firstID, opinionID, sectionID, 0, 1, 1, 1, "The Board denied relief. The court of appeals affirmed.", true)
	if err != nil {
		t.Fatalf("first passage: %v", err)
	}
	second, err := buildPassageFromSourceText(secondID, opinionID, sectionID, 2, 3, 1, 1, "The agency applied the wrong standard. The statute points the other way.", true)
	if err != nil {
		t.Fatalf("second passage: %v", err)
	}
	third, err := buildPassageFromSourceText(thirdID, opinionID, sectionID, 4, 4, 2, 2, "Substantial evidence controls.", true)
	if err != nil {
		t.Fatalf("third passage: %v", err)
	}

	snapshot, err := NewPassageRepairSnapshot("repair-session", 0, opinionID, []Passage{first, second, third}, nil)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	return snapshot
}
