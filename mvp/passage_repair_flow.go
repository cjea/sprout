package mvp

import "errors"

var (
	ErrPassageRepairCaseFocusMissing = errors.New("passage repair case focus is missing")
)

type PassageRepairFlowStage string

const (
	PassageRepairFlowInspectContext PassageRepairFlowStage = "inspect_context"
	PassageRepairFlowApplyOperation PassageRepairFlowStage = "apply_operation"
	PassageRepairFlowReviewResult   PassageRepairFlowStage = "review_result"
)

type PassageRepairCase struct {
	SessionID      string
	FocusPassageID PassageID
	Stage          PassageRepairFlowStage
	Snapshot       PassageRepairSnapshot
	Issues         []PassageIssue
}

func OpenPassageRepairCase(snapshot PassageRepairSnapshot, focus PassageID) (PassageRepairCase, error) {
	if err := snapshot.Validate(); err != nil {
		return PassageRepairCase{}, err
	}
	if _, err := passageIndex(snapshot.Passages, focus); err != nil {
		return PassageRepairCase{}, ErrPassageRepairCaseFocusMissing
	}
	return PassageRepairCase{
		SessionID:      snapshot.SessionID,
		FocusPassageID: focus,
		Stage:          PassageRepairFlowInspectContext,
		Snapshot:       snapshot,
	}, nil
}

func ClassifyPassageRepairCase(repairCase PassageRepairCase, issues []PassageIssue) PassageRepairCase {
	repairCase.Issues = append([]PassageIssue(nil), issues...)
	repairCase.Stage = PassageRepairFlowApplyOperation
	return repairCase
}

func ApplyPassageRepairCaseOperation(session *PassageRepairSession, repairCase PassageRepairCase, operation AdminPassageOperation) (PassageRepairCase, error) {
	if err := session.Apply(operation); err != nil {
		return PassageRepairCase{}, err
	}
	repairCase.Snapshot = session.Current
	repairCase.Stage = PassageRepairFlowReviewResult
	return repairCase, nil
}

func UndoPassageRepairCaseOperation(session *PassageRepairSession, repairCase PassageRepairCase) (PassageRepairCase, error) {
	if err := session.Undo(); err != nil {
		return PassageRepairCase{}, err
	}
	repairCase.Snapshot = session.Current
	repairCase.Stage = PassageRepairFlowReviewResult
	return repairCase, nil
}
