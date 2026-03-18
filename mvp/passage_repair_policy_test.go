package mvp

import (
	"errors"
	"testing"
)

func TestParseAdminPassageOperationKindRejectsTextEditing(t *testing.T) {
	testCases := []string{
		"edit_text",
		"replace_text",
		"rewrite_passage",
	}

	for _, testCase := range testCases {
		t.Run(testCase, func(t *testing.T) {
			_, err := ParseAdminPassageOperationKind(testCase)
			if !errors.Is(err, ErrUnknownAdminPassageOperation) {
				t.Fatalf("got err %v want %v", err, ErrUnknownAdminPassageOperation)
			}
		})
	}
}

func TestNewAdminPassageOperationAcceptsOnlyStructuralOperations(t *testing.T) {
	opinionID, _ := NewOpinionID("24-777_9ol1")
	firstPassageID, _ := NewPassageID("syllabus-1")
	secondPassageID, _ := NewPassageID("syllabus-2")

	target, err := NewPassageRepairTarget(opinionID, []PassageID{firstPassageID, secondPassageID})
	if err != nil {
		t.Fatalf("new target: %v", err)
	}

	merge, err := NewAdminPassageOperation(AdminPassageOperationMergeWithNext, target, nil)
	if err != nil {
		t.Fatalf("new merge operation: %v", err)
	}
	if merge.Kind != AdminPassageOperationMergeWithNext {
		t.Fatalf("got kind %q", merge.Kind)
	}

	splitAfter := SentenceNo(0)
	split, err := NewAdminPassageOperation(AdminPassageOperationSplitAtSentence, target, &splitAfter)
	if err != nil {
		t.Fatalf("new split operation: %v", err)
	}
	if split.SplitAfterSentence == nil || *split.SplitAfterSentence != 0 {
		t.Fatalf("got split boundary %#v", split.SplitAfterSentence)
	}
}

func TestNewAdminPassageOperationRejectsInvalidSplitShape(t *testing.T) {
	opinionID, _ := NewOpinionID("24-777_9ol1")
	passageID, _ := NewPassageID("syllabus-1")

	target, err := NewPassageRepairTarget(opinionID, []PassageID{passageID})
	if err != nil {
		t.Fatalf("new target: %v", err)
	}

	_, err = NewAdminPassageOperation(AdminPassageOperationSplitAtSentence, target, nil)
	if !errors.Is(err, ErrInvalidSentenceRange) {
		t.Fatalf("got err %v want %v", err, ErrInvalidSentenceRange)
	}

	splitAfter := SentenceNo(0)
	_, err = NewAdminPassageOperation(AdminPassageOperationMergeWithNext, target, &splitAfter)
	if !errors.Is(err, ErrInvalidSentenceRange) {
		t.Fatalf("got err %v want %v", err, ErrInvalidSentenceRange)
	}
}

func TestNewPassageRepairTargetRequiresSourceDerivedPassages(t *testing.T) {
	opinionID, _ := NewOpinionID("24-777_9ol1")
	passageID, _ := NewPassageID("syllabus-1")

	_, err := NewPassageRepairTarget("", []PassageID{passageID})
	if !errors.Is(err, ErrInvalidPassageRepairTarget) {
		t.Fatalf("got err %v want %v", err, ErrInvalidPassageRepairTarget)
	}

	_, err = NewPassageRepairTarget(opinionID, nil)
	if !errors.Is(err, ErrInvalidPassageRepairTarget) {
		t.Fatalf("got err %v want %v", err, ErrInvalidPassageRepairTarget)
	}
}

func TestNewSystemPassageTransformEnforcesGuessPrefixForProbabilisticTransforms(t *testing.T) {
	opinionID, _ := NewOpinionID("24-777_9ol1")
	passageID, _ := NewPassageID("syllabus-1")

	target, err := NewPassageRepairTarget(opinionID, []PassageID{passageID})
	if err != nil {
		t.Fatalf("new target: %v", err)
	}

	transform, err := NewSystemPassageTransform(SystemPassageTransformGuessSentenceRepair, target, true)
	if err != nil {
		t.Fatalf("new guess transform: %v", err)
	}
	if transform.Kind != SystemPassageTransformGuessSentenceRepair {
		t.Fatalf("got kind %q", transform.Kind)
	}

	_, err = NewSystemPassageTransform(SystemPassageTransformRepairHyphenation, target, true)
	if !errors.Is(err, ErrGuessTransformNameRequired) {
		t.Fatalf("got err %v want %v", err, ErrGuessTransformNameRequired)
	}

	_, err = NewSystemPassageTransform(SystemPassageTransformGuessIssueClass, target, false)
	if !errors.Is(err, ErrUnexpectedTransformGuessPrefix) {
		t.Fatalf("got err %v want %v", err, ErrUnexpectedTransformGuessPrefix)
	}
}

func TestParseSystemPassageTransformKindRejectsUnknownTransform(t *testing.T) {
	_, err := ParseSystemPassageTransformKind("rewrite_source_text")
	if !errors.Is(err, ErrUnknownSystemPassageTransform) {
		t.Fatalf("got err %v want %v", err, ErrUnknownSystemPassageTransform)
	}
}
