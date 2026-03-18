package mvp

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

var (
	ErrPassageRepairPassageMissing        = errors.New("passage repair passage is missing")
	ErrPassageRepairDroppedPassageMissing = errors.New("dropped passage is missing")
	ErrPassageRepairNoAdjacentPassage     = errors.New("adjacent passage is required")
	ErrPassageRepairSplitBoundaryInvalid  = errors.New("split boundary is invalid")
	ErrPassageRepairNoHistory             = errors.New("passage repair history is empty")
	ErrPassageRepairRevisionInvalid       = errors.New("passage repair revision is invalid")
)

type DroppedPassage struct {
	Passage       Passage
	OriginalIndex int
}

type PassageRepairSnapshot struct {
	SessionID string
	Revision  int
	OpinionID OpinionID
	Passages  []Passage
	Dropped   []DroppedPassage
}

func NewPassageRepairSnapshot(sessionID string, revision int, opinionID OpinionID, passages []Passage, dropped []DroppedPassage) (PassageRepairSnapshot, error) {
	snapshot := PassageRepairSnapshot{
		SessionID: strings.TrimSpace(sessionID),
		Revision:  revision,
		OpinionID: opinionID,
		Passages:  clonePassages(passages),
		Dropped:   cloneDroppedPassages(dropped),
	}
	if err := snapshot.Validate(); err != nil {
		return PassageRepairSnapshot{}, err
	}
	return snapshot, nil
}

func (s PassageRepairSnapshot) Validate() error {
	if strings.TrimSpace(s.SessionID) == "" {
		return ErrEmptyOpinionID
	}
	if s.Revision < 0 {
		return ErrPassageRepairRevisionInvalid
	}
	if strings.TrimSpace(string(s.OpinionID)) == "" {
		return ErrEmptyOpinionID
	}
	for _, passage := range s.Passages {
		if err := passage.Validate(); err != nil {
			return err
		}
		if passage.OpinionID != s.OpinionID {
			return ErrPassageOpinionMismatch
		}
	}
	for _, dropped := range s.Dropped {
		if err := dropped.Passage.Validate(); err != nil {
			return err
		}
		if dropped.OriginalIndex < 0 {
			return ErrPassageRepairRevisionInvalid
		}
	}
	return nil
}

type PassageRepairHistoryEntry struct {
	Revision  int
	Operation AdminPassageOperation
	Before    PassageRepairSnapshot
	After     PassageRepairSnapshot
}

type PassageRepairSession struct {
	Current PassageRepairSnapshot
	History []PassageRepairHistoryEntry
}

func NewPassageRepairSession(snapshot PassageRepairSnapshot) (*PassageRepairSession, error) {
	if err := snapshot.Validate(); err != nil {
		return nil, err
	}
	return &PassageRepairSession{Current: snapshot}, nil
}

func (s *PassageRepairSession) Apply(operation AdminPassageOperation) error {
	if s == nil {
		return ErrPassageRepairRevisionInvalid
	}
	next, err := ApplyAdminPassageOperation(s.Current, operation)
	if err != nil {
		return err
	}
	entry := PassageRepairHistoryEntry{
		Revision:  next.Revision,
		Operation: operation,
		Before:    s.Current,
		After:     next,
	}
	s.History = append(s.History, entry)
	s.Current = next
	return nil
}

func (s *PassageRepairSession) Undo() error {
	if s == nil || len(s.History) == 0 {
		return ErrPassageRepairNoHistory
	}
	last := s.History[len(s.History)-1]
	s.Current = last.Before
	s.History = s.History[:len(s.History)-1]
	return nil
}

func ApplyAdminPassageOperation(snapshot PassageRepairSnapshot, operation AdminPassageOperation) (PassageRepairSnapshot, error) {
	if err := snapshot.Validate(); err != nil {
		return PassageRepairSnapshot{}, err
	}
	if err := operation.Validate(); err != nil {
		return PassageRepairSnapshot{}, err
	}

	switch operation.Kind {
	case AdminPassageOperationMergeWithNext:
		return applyMergeWithNext(snapshot, operation.Target.PassageIDs[0])
	case AdminPassageOperationMergeWithPrevious:
		return applyMergeWithPrevious(snapshot, operation.Target.PassageIDs[0])
	case AdminPassageOperationSplitAtSentence:
		return applySplitAtSentence(snapshot, operation.Target.PassageIDs[0], *operation.SplitAfterSentence)
	case AdminPassageOperationMoveLastSentenceNext:
		return applyMoveLastSentenceToNext(snapshot, operation.Target.PassageIDs[0])
	case AdminPassageOperationMoveFirstSentencePrev:
		return applyMoveFirstSentenceToPrevious(snapshot, operation.Target.PassageIDs[0])
	case AdminPassageOperationDropPassage:
		return applyDropPassage(snapshot, operation.Target.PassageIDs[0])
	case AdminPassageOperationRestorePassage:
		return applyRestorePassage(snapshot, operation.Target.PassageIDs[0])
	case AdminPassageOperationRemoveRunningHeader:
		return applyRemoveRunningHeader(snapshot, operation.Target.PassageIDs[0])
	default:
		return PassageRepairSnapshot{}, ErrUnknownAdminPassageOperation
	}
}

func applyMergeWithNext(snapshot PassageRepairSnapshot, passageID PassageID) (PassageRepairSnapshot, error) {
	index, err := passageIndex(snapshot.Passages, passageID)
	if err != nil {
		return PassageRepairSnapshot{}, err
	}
	if index+1 >= len(snapshot.Passages) {
		return PassageRepairSnapshot{}, ErrPassageRepairNoAdjacentPassage
	}
	merged, err := mergePassages(snapshot.Passages[index], snapshot.Passages[index+1], snapshot.Passages[index].PassageID)
	if err != nil {
		return PassageRepairSnapshot{}, err
	}
	passages := clonePassages(snapshot.Passages)
	passages[index] = merged
	passages = append(passages[:index+1], passages[index+2:]...)
	return nextSnapshot(snapshot, passages, snapshot.Dropped)
}

func applyMergeWithPrevious(snapshot PassageRepairSnapshot, passageID PassageID) (PassageRepairSnapshot, error) {
	index, err := passageIndex(snapshot.Passages, passageID)
	if err != nil {
		return PassageRepairSnapshot{}, err
	}
	if index == 0 {
		return PassageRepairSnapshot{}, ErrPassageRepairNoAdjacentPassage
	}
	merged, err := mergePassages(snapshot.Passages[index-1], snapshot.Passages[index], snapshot.Passages[index-1].PassageID)
	if err != nil {
		return PassageRepairSnapshot{}, err
	}
	passages := clonePassages(snapshot.Passages)
	passages[index-1] = merged
	passages = append(passages[:index], passages[index+1:]...)
	return nextSnapshot(snapshot, passages, snapshot.Dropped)
}

func applySplitAtSentence(snapshot PassageRepairSnapshot, passageID PassageID, splitAfter SentenceNo) (PassageRepairSnapshot, error) {
	index, err := passageIndex(snapshot.Passages, passageID)
	if err != nil {
		return PassageRepairSnapshot{}, err
	}
	passage := snapshot.Passages[index]
	if splitAfter < passage.SentenceStart || splitAfter >= passage.SentenceEnd {
		return PassageRepairSnapshot{}, ErrPassageRepairSplitBoundaryInvalid
	}
	sentences := splitSentences(cleanPassageSourceText(string(passage.Text)))
	relative := int(splitAfter - passage.SentenceStart)
	if relative < 0 || relative >= len(sentences)-1 {
		return PassageRepairSnapshot{}, ErrPassageRepairSplitBoundaryInvalid
	}

	leftText := strings.Join(sentences[:relative+1], " ")
	rightText := strings.Join(sentences[relative+1:], " ")
	left, err := buildPassageFromSourceText(
		passage.PassageID,
		passage.OpinionID,
		passage.SectionID,
		passage.SentenceStart,
		splitAfter,
		passage.PageStart,
		passage.PageEnd,
		leftText,
		passage.FitsOnScreen,
	)
	if err != nil {
		return PassageRepairSnapshot{}, err
	}
	rightID, err := NewPassageID(fmt.Sprintf("%s-r", passage.PassageID))
	if err != nil {
		return PassageRepairSnapshot{}, err
	}
	right, err := buildPassageFromSourceText(
		rightID,
		passage.OpinionID,
		passage.SectionID,
		splitAfter+1,
		passage.SentenceEnd,
		passage.PageStart,
		passage.PageEnd,
		rightText,
		passage.FitsOnScreen,
	)
	if err != nil {
		return PassageRepairSnapshot{}, err
	}

	passages := clonePassages(snapshot.Passages)
	passages[index] = left
	passages = append(passages[:index+1], append([]Passage{right}, passages[index+1:]...)...)
	return nextSnapshot(snapshot, passages, snapshot.Dropped)
}

func applyMoveLastSentenceToNext(snapshot PassageRepairSnapshot, passageID PassageID) (PassageRepairSnapshot, error) {
	index, err := passageIndex(snapshot.Passages, passageID)
	if err != nil {
		return PassageRepairSnapshot{}, err
	}
	if index+1 >= len(snapshot.Passages) {
		return PassageRepairSnapshot{}, ErrPassageRepairNoAdjacentPassage
	}
	current := snapshot.Passages[index]
	next := snapshot.Passages[index+1]
	currentSentences := splitSentences(cleanPassageSourceText(string(current.Text)))
	if len(currentSentences) < 2 {
		return PassageRepairSnapshot{}, ErrPassageRepairSplitBoundaryInvalid
	}
	moved := currentSentences[len(currentSentences)-1]
	currentText := strings.Join(currentSentences[:len(currentSentences)-1], " ")
	nextText := strings.TrimSpace(moved + " " + string(next.Text))

	updatedCurrent, err := buildPassageFromSourceText(current.PassageID, current.OpinionID, current.SectionID, current.SentenceStart, current.SentenceEnd-1, current.PageStart, current.PageEnd, currentText, current.FitsOnScreen)
	if err != nil {
		return PassageRepairSnapshot{}, err
	}
	updatedNext, err := buildPassageFromSourceText(next.PassageID, next.OpinionID, next.SectionID, next.SentenceStart-1, next.SentenceEnd, minPage(current.PageStart, next.PageStart), maxPage(current.PageEnd, next.PageEnd), nextText, next.FitsOnScreen)
	if err != nil {
		return PassageRepairSnapshot{}, err
	}

	passages := clonePassages(snapshot.Passages)
	passages[index] = updatedCurrent
	passages[index+1] = updatedNext
	return nextSnapshot(snapshot, passages, snapshot.Dropped)
}

func applyMoveFirstSentenceToPrevious(snapshot PassageRepairSnapshot, passageID PassageID) (PassageRepairSnapshot, error) {
	index, err := passageIndex(snapshot.Passages, passageID)
	if err != nil {
		return PassageRepairSnapshot{}, err
	}
	if index == 0 {
		return PassageRepairSnapshot{}, ErrPassageRepairNoAdjacentPassage
	}
	current := snapshot.Passages[index]
	previous := snapshot.Passages[index-1]
	currentSentences := splitSentences(cleanPassageSourceText(string(current.Text)))
	if len(currentSentences) < 2 {
		return PassageRepairSnapshot{}, ErrPassageRepairSplitBoundaryInvalid
	}
	moved := currentSentences[0]
	currentText := strings.Join(currentSentences[1:], " ")
	previousText := strings.TrimSpace(string(previous.Text) + " " + moved)

	updatedPrevious, err := buildPassageFromSourceText(previous.PassageID, previous.OpinionID, previous.SectionID, previous.SentenceStart, previous.SentenceEnd+1, minPage(previous.PageStart, current.PageStart), maxPage(previous.PageEnd, current.PageEnd), previousText, previous.FitsOnScreen)
	if err != nil {
		return PassageRepairSnapshot{}, err
	}
	updatedCurrent, err := buildPassageFromSourceText(current.PassageID, current.OpinionID, current.SectionID, current.SentenceStart+1, current.SentenceEnd, current.PageStart, current.PageEnd, currentText, current.FitsOnScreen)
	if err != nil {
		return PassageRepairSnapshot{}, err
	}

	passages := clonePassages(snapshot.Passages)
	passages[index-1] = updatedPrevious
	passages[index] = updatedCurrent
	return nextSnapshot(snapshot, passages, snapshot.Dropped)
}

func applyDropPassage(snapshot PassageRepairSnapshot, passageID PassageID) (PassageRepairSnapshot, error) {
	index, err := passageIndex(snapshot.Passages, passageID)
	if err != nil {
		return PassageRepairSnapshot{}, err
	}
	passages := clonePassages(snapshot.Passages)
	dropped := cloneDroppedPassages(snapshot.Dropped)
	dropped = append(dropped, DroppedPassage{Passage: passages[index], OriginalIndex: index})
	passages = append(passages[:index], passages[index+1:]...)
	return nextSnapshot(snapshot, passages, dropped)
}

func applyRestorePassage(snapshot PassageRepairSnapshot, passageID PassageID) (PassageRepairSnapshot, error) {
	droppedIndex := -1
	var droppedPassage DroppedPassage
	for index, candidate := range snapshot.Dropped {
		if candidate.Passage.PassageID == passageID {
			droppedIndex = index
			droppedPassage = candidate
			break
		}
	}
	if droppedIndex < 0 {
		return PassageRepairSnapshot{}, ErrPassageRepairDroppedPassageMissing
	}

	passages := clonePassages(snapshot.Passages)
	insertAt := droppedPassage.OriginalIndex
	if insertAt > len(passages) {
		insertAt = len(passages)
	}
	passages = append(passages[:insertAt], append([]Passage{droppedPassage.Passage}, passages[insertAt:]...)...)
	dropped := cloneDroppedPassages(snapshot.Dropped)
	dropped = append(dropped[:droppedIndex], dropped[droppedIndex+1:]...)
	return nextSnapshot(snapshot, passages, dropped)
}

func applyRemoveRunningHeader(snapshot PassageRepairSnapshot, passageID PassageID) (PassageRepairSnapshot, error) {
	index, err := passageIndex(snapshot.Passages, passageID)
	if err != nil {
		return PassageRepairSnapshot{}, err
	}
	passage := snapshot.Passages[index]
	cleanedText := cleanPassageSourceText(string(passage.Text))
	updated, err := buildPassageFromSourceText(
		passage.PassageID,
		passage.OpinionID,
		passage.SectionID,
		passage.SentenceStart,
		passage.SentenceEnd,
		passage.PageStart,
		passage.PageEnd,
		cleanedText,
		passage.FitsOnScreen,
	)
	if err != nil {
		return PassageRepairSnapshot{}, err
	}
	passages := clonePassages(snapshot.Passages)
	passages[index] = updated
	return nextSnapshot(snapshot, passages, snapshot.Dropped)
}

func nextSnapshot(snapshot PassageRepairSnapshot, passages []Passage, dropped []DroppedPassage) (PassageRepairSnapshot, error) {
	ordered := clonePassages(passages)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].PageStart != ordered[j].PageStart {
			return ordered[i].PageStart < ordered[j].PageStart
		}
		if ordered[i].SentenceStart != ordered[j].SentenceStart {
			return ordered[i].SentenceStart < ordered[j].SentenceStart
		}
		return ordered[i].PassageID < ordered[j].PassageID
	})
	return NewPassageRepairSnapshot(snapshot.SessionID, snapshot.Revision+1, snapshot.OpinionID, ordered, dropped)
}

func buildPassageFromSourceText(
	passageID PassageID,
	opinionID OpinionID,
	sectionID SectionID,
	sentenceStart, sentenceEnd SentenceNo,
	pageStart, pageEnd PageNo,
	text string,
	fitsOnScreen bool,
) (Passage, error) {
	passage, err := NewPassage(passageID, opinionID, sectionID, sentenceStart, sentenceEnd, pageStart, pageEnd, Text(text), nil, fitsOnScreen)
	if err != nil {
		return Passage{}, err
	}
	citations, err := ExtractCitations(passage)
	if err != nil {
		return Passage{}, err
	}
	citations, err = NormalizeCitations(citations)
	if err != nil {
		return Passage{}, err
	}
	return NewPassage(passage.PassageID, passage.OpinionID, passage.SectionID, passage.SentenceStart, passage.SentenceEnd, passage.PageStart, passage.PageEnd, passage.Text, citations, passage.FitsOnScreen)
}

func mergePassages(left Passage, right Passage, mergedID PassageID) (Passage, error) {
	text := strings.TrimSpace(string(left.Text) + " " + string(right.Text))
	return buildPassageFromSourceText(
		mergedID,
		left.OpinionID,
		left.SectionID,
		left.SentenceStart,
		right.SentenceEnd,
		minPage(left.PageStart, right.PageStart),
		maxPage(left.PageEnd, right.PageEnd),
		text,
		left.FitsOnScreen && right.FitsOnScreen,
	)
}

func passageIndex(passages []Passage, passageID PassageID) (int, error) {
	for index, passage := range passages {
		if passage.PassageID == passageID {
			return index, nil
		}
	}
	return -1, ErrPassageRepairPassageMissing
}

func clonePassages(passages []Passage) []Passage {
	cloned := make([]Passage, 0, len(passages))
	for _, passage := range passages {
		cloned = append(cloned, passage)
	}
	return cloned
}

func cloneDroppedPassages(dropped []DroppedPassage) []DroppedPassage {
	cloned := make([]DroppedPassage, 0, len(dropped))
	for _, item := range dropped {
		cloned = append(cloned, item)
	}
	return cloned
}

func minPage(left, right PageNo) PageNo {
	if left < right {
		return left
	}
	return right
}

func maxPage(left, right PageNo) PageNo {
	if left > right {
		return left
	}
	return right
}
