package mvp

import (
	"testing"
)

func TestOpenPassageRepairCase(t *testing.T) {
	snapshot := newRepairSnapshotFixture(t)

	repairCase, err := OpenPassageRepairCase(snapshot, snapshot.Passages[0].PassageID)
	if err != nil {
		t.Fatalf("open repair case: %v", err)
	}
	if repairCase.Stage != PassageRepairFlowInspectContext {
		t.Fatalf("got stage %q", repairCase.Stage)
	}
}

func TestClassifyPassageRepairCase(t *testing.T) {
	snapshot := newRepairSnapshotFixture(t)
	repairCase, err := OpenPassageRepairCase(snapshot, snapshot.Passages[0].PassageID)
	if err != nil {
		t.Fatalf("open repair case: %v", err)
	}

	issue, err := NewPassageIssue(PassageIssueCitationDetached, snapshot.Passages[0].PassageID, "next passage begins with a cite")
	if err != nil {
		t.Fatalf("new issue: %v", err)
	}
	classified := ClassifyPassageRepairCase(repairCase, []PassageIssue{issue})
	if classified.Stage != PassageRepairFlowApplyOperation {
		t.Fatalf("got stage %q", classified.Stage)
	}
	if len(classified.Issues) != 1 {
		t.Fatalf("got %d issues want 1", len(classified.Issues))
	}
}

func TestApplyAndUndoPassageRepairCaseOperation(t *testing.T) {
	snapshot := newRepairSnapshotFixture(t)
	session, err := NewPassageRepairSession(snapshot)
	if err != nil {
		t.Fatalf("new session: %v", err)
	}
	repairCase, err := OpenPassageRepairCase(snapshot, snapshot.Passages[0].PassageID)
	if err != nil {
		t.Fatalf("open repair case: %v", err)
	}

	target, _ := NewPassageRepairTarget(snapshot.OpinionID, []PassageID{snapshot.Passages[0].PassageID})
	operation, _ := NewAdminPassageOperation(AdminPassageOperationMergeWithNext, target, nil)
	applied, err := ApplyPassageRepairCaseOperation(session, repairCase, operation)
	if err != nil {
		t.Fatalf("apply operation: %v", err)
	}
	if applied.Stage != PassageRepairFlowReviewResult {
		t.Fatalf("got stage %q", applied.Stage)
	}
	if len(applied.Snapshot.Passages) != 2 {
		t.Fatalf("got %d passages want 2", len(applied.Snapshot.Passages))
	}

	undone, err := UndoPassageRepairCaseOperation(session, applied)
	if err != nil {
		t.Fatalf("undo operation: %v", err)
	}
	if undone.Stage != PassageRepairFlowReviewResult {
		t.Fatalf("got stage %q", undone.Stage)
	}
	if len(undone.Snapshot.Passages) != 3 {
		t.Fatalf("got %d passages want 3", len(undone.Snapshot.Passages))
	}
}
