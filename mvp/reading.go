package mvp

import "errors"

var (
	ErrEmptyQueue       = errors.New("queue must contain at least one pending passage")
	ErrDuplicatePassage = errors.New("queue contains duplicate passage ids")
	ErrProgressPassage  = errors.New("current passage must be present when progress is updated")
	ErrTrailOrigin      = errors.New("trail origin passage is required")
	ErrTrailActive      = errors.New("trail active passage is required")
)

type Queue struct {
	UserID    UserID
	OpinionID OpinionID
	Pending   []PassageID
}

func NewQueue(userID UserID, opinionID OpinionID, pending []PassageID) (Queue, error) {
	queue := Queue{
		UserID:    userID,
		OpinionID: opinionID,
		Pending:   append([]PassageID(nil), pending...),
	}
	if err := queue.Validate(); err != nil {
		return Queue{}, err
	}
	return queue, nil
}

func (q Queue) Validate() error {
	if string(q.UserID) == "" {
		return ErrEmptyUserID
	}
	if string(q.OpinionID) == "" {
		return ErrEmptyOpinionID
	}
	if len(q.Pending) == 0 {
		return ErrEmptyQueue
	}

	seen := map[PassageID]struct{}{}
	for _, passageID := range q.Pending {
		if string(passageID) == "" {
			return ErrEmptyPassageID
		}
		if _, ok := seen[passageID]; ok {
			return ErrDuplicatePassage
		}
		seen[passageID] = struct{}{}
	}
	return nil
}

type Progress struct {
	UserID            UserID
	OpinionID         OpinionID
	CurrentPassage    *PassageID
	CompletedPassages []PassageID
	OpenQuestionIDs   []QuestionID
	UpdatedAt         Timestamp
}

func NewProgress(userID UserID, opinionID OpinionID, currentPassage *PassageID, completedPassages []PassageID, openQuestionIDs []QuestionID, updatedAt Timestamp) (Progress, error) {
	progress := Progress{
		UserID:            userID,
		OpinionID:         opinionID,
		CurrentPassage:    currentPassage,
		CompletedPassages: append([]PassageID(nil), completedPassages...),
		OpenQuestionIDs:   append([]QuestionID(nil), openQuestionIDs...),
		UpdatedAt:         updatedAt,
	}
	if err := progress.Validate(); err != nil {
		return Progress{}, err
	}
	return progress, nil
}

func (p Progress) Validate() error {
	if string(p.UserID) == "" {
		return ErrEmptyUserID
	}
	if string(p.OpinionID) == "" {
		return ErrEmptyOpinionID
	}
	if p.CurrentPassage != nil && string(*p.CurrentPassage) == "" {
		return ErrProgressPassage
	}
	return nil
}

func (Progress) ProgressRecord() {}

type Trail struct {
	OriginPassage PassageID
	ActivePassage PassageID
	QuestionStack []QuestionID
}

func NewTrail(originPassage PassageID, activePassage PassageID, questionStack []QuestionID) (Trail, error) {
	trail := Trail{
		OriginPassage: originPassage,
		ActivePassage: activePassage,
		QuestionStack: append([]QuestionID(nil), questionStack...),
	}
	if err := trail.Validate(); err != nil {
		return Trail{}, err
	}
	return trail, nil
}

func (t Trail) Validate() error {
	switch {
	case string(t.OriginPassage) == "":
		return ErrTrailOrigin
	case string(t.ActivePassage) == "":
		return ErrTrailActive
	default:
		return nil
	}
}

type ReadingState struct {
	Opinion   Opinion
	Passage   Passage
	Citations []Citation
	Progress  Progress
	Trail     Trail
}

func NewReadingState(opinion Opinion, passage Passage, citations []Citation, progress Progress, trail Trail) (ReadingState, error) {
	state := ReadingState{
		Opinion:   opinion,
		Passage:   passage,
		Citations: append([]Citation(nil), citations...),
		Progress:  progress,
		Trail:     trail,
	}
	if err := state.Validate(); err != nil {
		return ReadingState{}, err
	}
	return state, nil
}

func (r ReadingState) Validate() error {
	if err := r.Opinion.Validate(); err != nil {
		return err
	}
	if err := r.Passage.Validate(); err != nil {
		return err
	}
	for _, citation := range r.Citations {
		if err := citation.Validate(); err != nil {
			return err
		}
	}
	if err := r.Progress.Validate(); err != nil {
		return err
	}
	return r.Trail.Validate()
}
