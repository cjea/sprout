package mvp

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrUnknownAdminPassageOperation   = errors.New("unknown admin passage operation")
	ErrUnknownSystemPassageTransform  = errors.New("unknown system passage transform")
	ErrInvalidPassageRepairTarget     = errors.New("passage repair target is invalid")
	ErrGuessTransformNameRequired     = errors.New("guess-backed transform names must start with guess")
	ErrUnexpectedTransformGuessPrefix = errors.New("non-probabilistic transform must not start with guess")
)

type AdminPassageOperationKind string

const (
	AdminPassageOperationMergeWithNext         AdminPassageOperationKind = "merge_with_next"
	AdminPassageOperationMergeWithPrevious     AdminPassageOperationKind = "merge_with_previous"
	AdminPassageOperationSplitAtSentence       AdminPassageOperationKind = "split_at_sentence"
	AdminPassageOperationMoveLastSentenceNext  AdminPassageOperationKind = "move_last_sentence_to_next"
	AdminPassageOperationMoveFirstSentencePrev AdminPassageOperationKind = "move_first_sentence_to_previous"
	AdminPassageOperationDropPassage           AdminPassageOperationKind = "drop_passage"
	AdminPassageOperationRestorePassage        AdminPassageOperationKind = "restore_passage"
	AdminPassageOperationRemoveRunningHeader   AdminPassageOperationKind = "remove_running_header"
	AdminPassageOperationUndo                  AdminPassageOperationKind = "revert_last_operation"
)

func ParseAdminPassageOperationKind(value string) (AdminPassageOperationKind, error) {
	switch AdminPassageOperationKind(strings.TrimSpace(value)) {
	case AdminPassageOperationMergeWithNext,
		AdminPassageOperationMergeWithPrevious,
		AdminPassageOperationSplitAtSentence,
		AdminPassageOperationMoveLastSentenceNext,
		AdminPassageOperationMoveFirstSentencePrev,
		AdminPassageOperationDropPassage,
		AdminPassageOperationRestorePassage,
		AdminPassageOperationRemoveRunningHeader,
		AdminPassageOperationUndo:
		return AdminPassageOperationKind(strings.TrimSpace(value)), nil
	default:
		return "", ErrUnknownAdminPassageOperation
	}
}

type SystemPassageTransformKind string

const (
	SystemPassageTransformRemoveRunningHeaders      SystemPassageTransformKind = "remove_running_headers"
	SystemPassageTransformRepairHyphenation         SystemPassageTransformKind = "repair_hyphenation_artifacts"
	SystemPassageTransformRerunSentenceSegmentation SystemPassageTransformKind = "rerun_sentence_segmentation"
	SystemPassageTransformGuessSentenceRepair       SystemPassageTransformKind = "guessSentenceRepair"
	SystemPassageTransformGuessIssueClass           SystemPassageTransformKind = "guessIssueClass"
)

func ParseSystemPassageTransformKind(value string) (SystemPassageTransformKind, error) {
	switch SystemPassageTransformKind(strings.TrimSpace(value)) {
	case SystemPassageTransformRemoveRunningHeaders,
		SystemPassageTransformRepairHyphenation,
		SystemPassageTransformRerunSentenceSegmentation,
		SystemPassageTransformGuessSentenceRepair,
		SystemPassageTransformGuessIssueClass:
		return SystemPassageTransformKind(strings.TrimSpace(value)), nil
	default:
		return "", ErrUnknownSystemPassageTransform
	}
}

type PassageRepairTarget struct {
	OpinionID  OpinionID
	PassageIDs []PassageID
}

func NewPassageRepairTarget(opinionID OpinionID, passageIDs []PassageID) (PassageRepairTarget, error) {
	target := PassageRepairTarget{
		OpinionID:  opinionID,
		PassageIDs: append([]PassageID(nil), passageIDs...),
	}
	if err := target.Validate(); err != nil {
		return PassageRepairTarget{}, err
	}
	return target, nil
}

func (t PassageRepairTarget) Validate() error {
	if strings.TrimSpace(string(t.OpinionID)) == "" {
		return ErrInvalidPassageRepairTarget
	}
	if len(t.PassageIDs) == 0 {
		return ErrInvalidPassageRepairTarget
	}
	for _, passageID := range t.PassageIDs {
		if strings.TrimSpace(string(passageID)) == "" {
			return ErrInvalidPassageRepairTarget
		}
	}
	return nil
}

type AdminPassageOperation struct {
	Kind               AdminPassageOperationKind
	Target             PassageRepairTarget
	SplitAfterSentence *SentenceNo
}

func NewAdminPassageOperation(
	kind AdminPassageOperationKind,
	target PassageRepairTarget,
	splitAfterSentence *SentenceNo,
) (AdminPassageOperation, error) {
	operation := AdminPassageOperation{
		Kind:               kind,
		Target:             target,
		SplitAfterSentence: splitAfterSentence,
	}
	if err := operation.Validate(); err != nil {
		return AdminPassageOperation{}, err
	}
	return operation, nil
}

func (o AdminPassageOperation) Validate() error {
	if _, err := ParseAdminPassageOperationKind(string(o.Kind)); err != nil {
		return err
	}
	if err := o.Target.Validate(); err != nil {
		return err
	}
	if o.SplitAfterSentence != nil && *o.SplitAfterSentence < 0 {
		return ErrInvalidSentenceRange
	}
	switch o.Kind {
	case AdminPassageOperationSplitAtSentence:
		if o.SplitAfterSentence == nil {
			return ErrInvalidSentenceRange
		}
	default:
		if o.SplitAfterSentence != nil {
			return ErrInvalidSentenceRange
		}
	}
	return nil
}

type SystemPassageTransform struct {
	Kind          SystemPassageTransformKind
	Target        PassageRepairTarget
	Probabilistic bool
}

func NewSystemPassageTransform(
	kind SystemPassageTransformKind,
	target PassageRepairTarget,
	probabilistic bool,
) (SystemPassageTransform, error) {
	transform := SystemPassageTransform{
		Kind:          kind,
		Target:        target,
		Probabilistic: probabilistic,
	}
	if err := transform.Validate(); err != nil {
		return SystemPassageTransform{}, err
	}
	return transform, nil
}

func (t SystemPassageTransform) Validate() error {
	if _, err := ParseSystemPassageTransformKind(string(t.Kind)); err != nil {
		return err
	}
	if err := t.Target.Validate(); err != nil {
		return err
	}
	name := string(t.Kind)
	hasGuessPrefix := strings.HasPrefix(name, "guess")
	switch {
	case t.Probabilistic && !hasGuessPrefix:
		return fmt.Errorf("%w: %s", ErrGuessTransformNameRequired, name)
	case !t.Probabilistic && hasGuessPrefix:
		return fmt.Errorf("%w: %s", ErrUnexpectedTransformGuessPrefix, name)
	default:
		return nil
	}
}
