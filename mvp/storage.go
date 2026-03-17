package mvp

import (
	"errors"
	"sync"
)

var (
	ErrNotFound            = errors.New("record not found")
	ErrWrongRecordType     = errors.New("record has unexpected type")
	ErrQuestionSaveInvalid = errors.New("question record is invalid")
)

type MemoryStorage struct {
	mu        sync.RWMutex
	rawPDFs   map[OpinionID]RawPDF
	opinions  map[OpinionID]Opinion
	passages  map[PassageID]Passage
	progress  map[string]Progress
	questions map[string][]Question
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		rawPDFs:   map[OpinionID]RawPDF{},
		opinions:  map[OpinionID]Opinion{},
		passages:  map[PassageID]Passage{},
		progress:  map[string]Progress{},
		questions: map[string][]Question{},
	}
}

func (s *MemoryStorage) InTx(fn func(Storage) error) error {
	return fn(s)
}

func (s *MemoryStorage) SaveRawPDF(record RawPDFRecord) (RawPDFRecord, error) {
	raw, err := rawPDFFromRecord(record)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rawPDFs[raw.OpinionID] = raw
	return raw, nil
}

func (s *MemoryStorage) SaveOpinion(record OpinionRecord) (OpinionRecord, error) {
	opinion, err := opinionFromRecord(record)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.opinions[opinion.OpinionID] = opinion
	return opinion, nil
}

func (s *MemoryStorage) SavePassages(records []PassageRecord) ([]PassageRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	saved := make([]PassageRecord, 0, len(records))
	for _, record := range records {
		passage, err := passageFromRecord(record)
		if err != nil {
			return nil, err
		}
		s.passages[passage.PassageID] = passage
		saved = append(saved, passage)
	}
	return saved, nil
}

func (s *MemoryStorage) SaveProgress(record ProgressRecord) (ProgressRecord, error) {
	progress, err := progressFromRecord(record)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.progress[progressKey(progress.UserID, progress.OpinionID)] = progress
	return progress, nil
}

func (s *MemoryStorage) SaveQuestionRecord(record QuestionRecord) (QuestionRecord, error) {
	question, err := questionFromRecord(record)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	key := questionKey(question.UserID, question.Anchor.OpinionID)
	s.questions[key] = append(s.questions[key], question)
	return question, nil
}

func (s *MemoryStorage) LoadRawPDF(opinionID OpinionID) (RawPDFRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	raw, ok := s.rawPDFs[opinionID]
	if !ok {
		return nil, ErrNotFound
	}
	return raw, nil
}

func (s *MemoryStorage) LoadOpinion(opinionID OpinionID) (OpinionRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	opinion, ok := s.opinions[opinionID]
	if !ok {
		return nil, ErrNotFound
	}
	return opinion, nil
}

func (s *MemoryStorage) LoadPassage(passageID PassageID) (PassageRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	passage, ok := s.passages[passageID]
	if !ok {
		return nil, ErrNotFound
	}
	return passage, nil
}

func (s *MemoryStorage) LoadProgress(userID UserID, opinionID OpinionID) (ProgressRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	progress, ok := s.progress[progressKey(userID, opinionID)]
	if !ok {
		return nil, ErrNotFound
	}
	return progress, nil
}

func (s *MemoryStorage) LoadQuestions(userID UserID, opinionID OpinionID) ([]QuestionRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	questions, ok := s.questions[questionKey(userID, opinionID)]
	if !ok {
		return nil, ErrNotFound
	}
	records := make([]QuestionRecord, 0, len(questions))
	for _, question := range questions {
		records = append(records, question)
	}
	return records, nil
}

func (s *MemoryStorage) ListPassages(opinionID OpinionID) ([]PassageRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	records := make([]PassageRecord, 0)
	for _, passage := range s.passages {
		if passage.OpinionID != opinionID {
			continue
		}
		records = append(records, passage)
	}
	if len(records) == 0 {
		return nil, ErrNotFound
	}
	return records, nil
}

func progressKey(userID UserID, opinionID OpinionID) string {
	return string(userID) + ":" + string(opinionID)
}

func questionKey(userID UserID, opinionID OpinionID) string {
	return string(userID) + ":" + string(opinionID)
}

func rawPDFFromRecord(record RawPDFRecord) (RawPDF, error) {
	raw, ok := record.(RawPDF)
	if !ok {
		return RawPDF{}, ErrWrongRecordType
	}
	return raw, nil
}

func opinionFromRecord(record OpinionRecord) (Opinion, error) {
	opinion, ok := record.(Opinion)
	if !ok {
		return Opinion{}, ErrWrongRecordType
	}
	return opinion, nil
}

func passageFromRecord(record PassageRecord) (Passage, error) {
	passage, ok := record.(Passage)
	if !ok {
		return Passage{}, ErrWrongRecordType
	}
	return passage, nil
}

func progressFromRecord(record ProgressRecord) (Progress, error) {
	progress, ok := record.(Progress)
	if !ok {
		return Progress{}, ErrWrongRecordType
	}
	return progress, nil
}

func questionFromRecord(record QuestionRecord) (Question, error) {
	question, ok := record.(Question)
	if !ok {
		return Question{}, ErrWrongRecordType
	}
	return question, nil
}

func saveRawPDF(storage Storage, raw RawPDF) (RawPDF, error) {
	record, err := storage.SaveRawPDF(raw)
	if err != nil {
		return RawPDF{}, err
	}
	return rawPDFFromRecord(record)
}

func saveOpinion(storage Storage, opinion Opinion) (Opinion, error) {
	record, err := storage.SaveOpinion(opinion)
	if err != nil {
		return Opinion{}, err
	}
	return opinionFromRecord(record)
}

func savePassages(storage Storage, passages []Passage) ([]Passage, error) {
	records := make([]PassageRecord, 0, len(passages))
	for _, passage := range passages {
		records = append(records, passage)
	}
	savedRecords, err := storage.SavePassages(records)
	if err != nil {
		return nil, err
	}
	saved := make([]Passage, 0, len(savedRecords))
	for _, record := range savedRecords {
		passage, err := passageFromRecord(record)
		if err != nil {
			return nil, err
		}
		saved = append(saved, passage)
	}
	return saved, nil
}

func saveProgress(storage Storage, progress Progress) (Progress, error) {
	record, err := storage.SaveProgress(progress)
	if err != nil {
		return Progress{}, err
	}
	return progressFromRecord(record)
}

func saveQuestion(storage Storage, question Question) (Question, error) {
	record, err := storage.SaveQuestionRecord(question)
	if err != nil {
		return Question{}, err
	}
	return questionFromRecord(record)
}

func loadRawPDF(storage Storage, opinionID OpinionID) (RawPDF, error) {
	record, err := storage.LoadRawPDF(opinionID)
	if err != nil {
		return RawPDF{}, err
	}
	return rawPDFFromRecord(record)
}

func loadOpinion(storage Storage, opinionID OpinionID) (Opinion, error) {
	record, err := storage.LoadOpinion(opinionID)
	if err != nil {
		return Opinion{}, err
	}
	return opinionFromRecord(record)
}

func loadPassage(storage Storage, passageID PassageID) (Passage, error) {
	record, err := storage.LoadPassage(passageID)
	if err != nil {
		return Passage{}, err
	}
	return passageFromRecord(record)
}

func loadProgress(storage Storage, userID UserID, opinionID OpinionID) (Progress, error) {
	record, err := storage.LoadProgress(userID, opinionID)
	if err != nil {
		return Progress{}, err
	}
	return progressFromRecord(record)
}

func listPassages(storage Storage, opinionID OpinionID) ([]Passage, error) {
	lister, ok := storage.(PassageLister)
	if !ok {
		return nil, ErrWrongRecordType
	}
	records, err := lister.ListPassages(opinionID)
	if err != nil {
		return nil, err
	}
	passages := make([]Passage, 0, len(records))
	for _, record := range records {
		passage, err := passageFromRecord(record)
		if err != nil {
			return nil, err
		}
		passages = append(passages, passage)
	}
	return passages, nil
}

func loadQuestions(storage Storage, userID UserID, opinionID OpinionID) ([]Question, error) {
	records, err := storage.LoadQuestions(userID, opinionID)
	if err != nil {
		return nil, err
	}
	questions := make([]Question, 0, len(records))
	for _, record := range records {
		question, err := questionFromRecord(record)
		if err != nil {
			return nil, err
		}
		questions = append(questions, question)
	}
	return questions, nil
}
