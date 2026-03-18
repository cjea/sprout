package main

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"sprout/mvp"
)

//go:embed app
var appFS embed.FS

const (
	defaultFixturePath = "fixtures/scotus/24-777_9ol1.pdf"
	defaultFixtureURL  = "https://www.supremecourt.gov/opinions/25pdf/24-777_9ol1.pdf"
	defaultDBPath      = "var/browser.db"
	defaultUserID      = "demo"
	defaultModelName   = "heuristic-v1"
)

type serverConfig struct {
	DBPath      string
	FixturePath string
	FixtureURL  string
	UserID      string
	ModelName   string
}

type server struct {
	storage   *mvp.SQLiteStorage
	files     http.Handler
	userID    mvp.UserID
	opinionID mvp.OpinionID
	model     mvp.Model
	logger    *log.Logger
}

type readerResponse struct {
	Opinion   opinionDTO        `json:"opinion"`
	Passage   passageDTO        `json:"passage"`
	Progress  progressDTO       `json:"progress"`
	Passages  []passageListItem `json:"passages"`
	Questions []questionDTO     `json:"questions"`
	Repair    repairDTO         `json:"repair"`
}

type completeRequest struct {
	UserID    string `json:"userId"`
	OpinionID string `json:"opinionId"`
	PassageID string `json:"passageId"`
}

type questionRequest struct {
	UserID    string `json:"userId"`
	OpinionID string `json:"opinionId"`
	PassageID string `json:"passageId"`
	Start     int    `json:"start"`
	End       int    `json:"end"`
	Text      string `json:"text"`
}

type questionResponse struct {
	Question questionDTO `json:"question"`
	Answer   answerDTO   `json:"answer"`
}

type repairRequest struct {
	UserID    string `json:"userId"`
	OpinionID string `json:"opinionId"`
	PassageID string `json:"passageId"`
	Operation string `json:"operation"`
}

type repairUndoRequest struct {
	UserID    string `json:"userId"`
	OpinionID string `json:"opinionId"`
}

type opinionDTO struct {
	OpinionID string       `json:"opinionId"`
	CaseName  string       `json:"caseName"`
	Docket    string       `json:"docket"`
	DecidedOn string       `json:"decidedOn"`
	Term      string       `json:"term"`
	Sections  []sectionDTO `json:"sections"`
}

type sectionDTO struct {
	SectionID string `json:"sectionId"`
	Kind      string `json:"kind"`
	Title     string `json:"title"`
	StartPage int    `json:"startPage"`
	EndPage   int    `json:"endPage"`
}

type passageDTO struct {
	PassageID    string        `json:"passageId"`
	SectionID    string        `json:"sectionId"`
	PageStart    int           `json:"pageStart"`
	PageEnd      int           `json:"pageEnd"`
	Text         string        `json:"text"`
	FitsOnScreen bool          `json:"fitsOnScreen"`
	Citations    []citationDTO `json:"citations"`
}

type citationDTO struct {
	CitationID string  `json:"citationId"`
	Kind       string  `json:"kind"`
	RawText    string  `json:"rawText"`
	Normalized *string `json:"normalized,omitempty"`
}

type progressDTO struct {
	UserID            string   `json:"userId"`
	OpinionID         string   `json:"opinionId"`
	CurrentPassageID  string   `json:"currentPassageId"`
	CompletedPassages []string `json:"completedPassages"`
	OpenQuestionIDs   []string `json:"openQuestionIds"`
}

type passageListItem struct {
	PassageID string `json:"passageId"`
	SectionID string `json:"sectionId"`
	Label     string `json:"label"`
}

type questionDTO struct {
	QuestionID string `json:"questionId"`
	Text       string `json:"text"`
	Status     string `json:"status"`
	PassageID  string `json:"passageId"`
	Quote      string `json:"quote"`
}

type answerDTO struct {
	QuestionID string        `json:"questionId"`
	Answer     string        `json:"answer"`
	Evidence   []evidenceDTO `json:"evidence"`
	Caveats    []string      `json:"caveats"`
	ModelName  string        `json:"modelName"`
}

type evidenceDTO struct {
	Label     string `json:"label"`
	Quote     string `json:"quote"`
	PassageID string `json:"passageId"`
}

type repairDTO struct {
	Issues           []repairIssueDTO   `json:"issues"`
	History          []repairHistoryDTO `json:"history"`
	CanMergeNext     bool               `json:"canMergeNext"`
	CanMergePrevious bool               `json:"canMergePrevious"`
	CanSplitSentence bool               `json:"canSplitSentence"`
	CanRemoveHeader  bool               `json:"canRemoveHeader"`
}

type repairIssueDTO struct {
	Kind    string `json:"kind"`
	Summary string `json:"summary"`
}

type repairHistoryDTO struct {
	Revision      int    `json:"revision"`
	OperationKind string `json:"operationKind"`
	TargetPassage string `json:"targetPassage"`
	CreatedAt     string `json:"createdAt"`
}

type repairResponse struct {
	PassageID string `json:"passageId"`
	Revision  int    `json:"revision"`
}

func main() {
	cfg := serverConfig{}
	flag.StringVar(&cfg.DBPath, "db", defaultDBPath, "sqlite database path")
	flag.StringVar(&cfg.FixturePath, "fixture", defaultFixturePath, "local fixture pdf path")
	flag.StringVar(&cfg.FixtureURL, "fixture-url", defaultFixtureURL, "fixture source url")
	flag.StringVar(&cfg.UserID, "user", defaultUserID, "browser user id")
	flag.StringVar(&cfg.ModelName, "model", defaultModelName, "model name for guess functions")
	addr := flag.String("addr", ":8080", "http listen address")
	flag.Parse()

	closeFn, err := serve(*addr, cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer closeFn()
}

func serve(addr string, cfg serverConfig) (func() error, error) {
	handler, closeFn, err := newServer(cfg)
	if err != nil {
		return nil, err
	}
	fmt.Printf("sprout-web listening on http://localhost%s\n", strings.TrimPrefix(addr, "localhost"))
	return closeFn, http.ListenAndServe(addr, handler)
}

func newServer(cfg serverConfig) (http.Handler, func() error, error) {
	sub, err := fs.Sub(appFS, "app")
	if err != nil {
		return nil, nil, err
	}

	storage, userID, opinionID, err := prepareBrowserState(cfg)
	if err != nil {
		return nil, nil, err
	}
	model, err := mvp.NewModel(cfg.ModelName, 8192)
	if err != nil {
		_ = storage.Close()
		return nil, nil, err
	}

	app := &server{
		storage:   storage,
		files:     http.FileServer(http.FS(sub)),
		userID:    userID,
		opinionID: opinionID,
		model:     model,
		logger:    log.New(os.Stderr, "sprout-web ", log.LstdFlags),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/reader", app.handleReader)
	mux.HandleFunc("/api/complete", app.handleComplete)
	mux.HandleFunc("/api/question", app.handleQuestion)
	mux.HandleFunc("/api/repair/apply", app.handleRepairApply)
	mux.HandleFunc("/api/repair/undo", app.handleRepairUndo)
	mux.HandleFunc("/", app.handleApp)

	return mux, storage.Close, nil
}

func prepareBrowserState(cfg serverConfig) (*mvp.SQLiteStorage, mvp.UserID, mvp.OpinionID, error) {
	if cfg.DBPath == "" {
		cfg.DBPath = defaultDBPath
	}
	if cfg.FixturePath == "" {
		cfg.FixturePath = defaultFixturePath
	}
	if cfg.FixtureURL == "" {
		cfg.FixtureURL = defaultFixtureURL
	}
	if cfg.UserID == "" {
		cfg.UserID = defaultUserID
	}
	if cfg.ModelName == "" {
		cfg.ModelName = defaultModelName
	}

	userID, err := mvp.NewUserID(cfg.UserID)
	if err != nil {
		return nil, "", "", err
	}
	fixtureURL, err := mvp.EnterURL(cfg.FixtureURL)
	if err != nil {
		return nil, "", "", err
	}
	opinionID, err := mvp.MakeOpinionID(fixtureURL)
	if err != nil {
		return nil, "", "", err
	}
	model, err := mvp.NewModel(cfg.ModelName, 8192)
	if err != nil {
		return nil, "", "", err
	}

	if dir := filepath.Dir(cfg.DBPath); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, "", "", fmt.Errorf("create db directory: %w", err)
		}
	}
	storage, err := mvp.OpenSQLiteWithMigrations(cfg.DBPath, mvp.SystemClock{})
	if err != nil {
		return nil, "", "", err
	}

	if _, err := storage.LoadOpinion(opinionID); errors.Is(err, mvp.ErrNotFound) {
		bytes, readErr := os.ReadFile(cfg.FixturePath)
		if readErr != nil {
			_ = storage.Close()
			return nil, "", "", fmt.Errorf("read fixture: %w", readErr)
		}
		raw, rawErr := mvp.MakeRawPDF(opinionID, fixtureURL, bytes, mvp.SystemClock{}.Now())
		if rawErr != nil {
			_ = storage.Close()
			return nil, "", "", rawErr
		}
		firstPassageID, ingestErr := mvp.IngestRawPDF(userID, raw, storage, model)
		if ingestErr != nil {
			_ = storage.Close()
			return nil, "", "", ingestErr
		}
		if _, openErr := mvp.OpenPassage(userID, firstPassageID, storage); openErr != nil {
			_ = storage.Close()
			return nil, "", "", openErr
		}
	} else if err != nil {
		_ = storage.Close()
		return nil, "", "", err
	}

	if _, err := storage.LoadProgress(userID, opinionID); errors.Is(err, mvp.ErrNotFound) {
		passages, listErr := listPassages(storage, opinionID)
		if listErr != nil {
			_ = storage.Close()
			return nil, "", "", listErr
		}
		if len(passages) == 0 {
			_ = storage.Close()
			return nil, "", "", mvp.ErrNotFound
		}
		if _, openErr := mvp.OpenPassage(userID, passages[0].PassageID, storage); openErr != nil {
			_ = storage.Close()
			return nil, "", "", openErr
		}
	} else if err != nil {
		_ = storage.Close()
		return nil, "", "", err
	}

	return storage, userID, opinionID, nil
}

func (s *server) handleApp(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/", "/app.css", "/app.js":
		s.files.ServeHTTP(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (s *server) handleReader(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := s.userID
	if raw := strings.TrimSpace(r.URL.Query().Get("user")); raw != "" {
		parsed, err := mvp.NewUserID(raw)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		userID = parsed
	}

	opinionID := s.opinionID
	if raw := strings.TrimSpace(r.URL.Query().Get("opinion")); raw != "" {
		parsed, err := mvp.NewOpinionID(raw)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		opinionID = parsed
	}

	passages, err := listPassages(s.storage, opinionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var state mvp.ReadingState
	if raw := strings.TrimSpace(r.URL.Query().Get("passage")); raw != "" {
		passageID, err := mvp.NewPassageID(raw)
		if err == nil {
			state, err = mvp.OpenPassage(userID, passageID, s.storage)
			if err == nil {
				goto loaded
			}
		}
	}

	state, err = mvp.ResumeReading(userID, opinionID, s.storage)
	if errors.Is(err, mvp.ErrNotFound) && len(passages) > 0 {
		state, err = mvp.OpenPassage(userID, passages[0].PassageID, s.storage)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

loaded:
	questions, err := loadQuestions(s.storage, userID, opinionID)
	if errors.Is(err, mvp.ErrNotFound) {
		questions = nil
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := readerResponse{
		Opinion:   opinionData(state.Opinion),
		Passage:   passageData(state.Passage),
		Progress:  progressData(state.Progress),
		Passages:  passageListData(passages),
		Questions: questionData(questions),
		Repair:    repairData(s.storage, opinionID, userID, passages, state.Passage),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *server) handleRepairApply(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, r, http.StatusMethodNotAllowed, errors.New("method not allowed"))
		return
	}

	var request repairRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.writeError(w, r, http.StatusBadRequest, err)
		return
	}

	userID, opinionID, err := s.requestIdentity(request.UserID, request.OpinionID)
	if err != nil {
		s.writeError(w, r, http.StatusBadRequest, err)
		return
	}
	passageID, err := mvp.NewPassageID(request.PassageID)
	if err != nil {
		s.writeError(w, r, http.StatusBadRequest, err)
		return
	}
	operation, err := parseBrowserRepairOperation(request.Operation, s.storage, opinionID, passageID)
	if err != nil {
		s.writeError(w, r, http.StatusBadRequest, err)
		return
	}

	passages, err := listPassages(s.storage, opinionID)
	if err != nil {
		s.writeError(w, r, http.StatusInternalServerError, err)
		return
	}
	session, err := mvp.LoadOrStartAuditedPassageRepairSession(r.Context(), s.storage, opinionID, string(userID), "browser", passages)
	if err != nil {
		s.writeError(w, r, http.StatusInternalServerError, err)
		return
	}
	before := session.Current
	if err := session.Apply(operation); err != nil {
		s.writeError(w, r, http.StatusBadRequest, err)
		return
	}
	if err := replacePassages(s.storage, opinionID, session.Current.Passages); err != nil {
		s.writeError(w, r, http.StatusInternalServerError, err)
		return
	}
	if err := mvp.RecordAuditedPassageRepairOperation(r.Context(), s.storage, string(userID), "browser", operation, before, session.Current, time.Now().UTC()); err != nil {
		s.writeError(w, r, http.StatusInternalServerError, err)
		return
	}
	focusPassageID := browserRepairFocusPassageID(operation, before, session.Current)
	if _, err := mvp.OpenPassage(userID, focusPassageID, s.storage); err != nil {
		s.writeError(w, r, http.StatusInternalServerError, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(repairResponse{
		PassageID: string(focusPassageID),
		Revision:  session.Current.Revision,
	})
}

func (s *server) handleRepairUndo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, r, http.StatusMethodNotAllowed, errors.New("method not allowed"))
		return
	}

	var request repairUndoRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.writeError(w, r, http.StatusBadRequest, err)
		return
	}
	userID, opinionID, err := s.requestIdentity(request.UserID, request.OpinionID)
	if err != nil {
		s.writeError(w, r, http.StatusBadRequest, err)
		return
	}

	passages, err := listPassages(s.storage, opinionID)
	if err != nil {
		s.writeError(w, r, http.StatusInternalServerError, err)
		return
	}
	session, err := mvp.LoadOrStartAuditedPassageRepairSession(r.Context(), s.storage, opinionID, string(userID), "browser", passages)
	if err != nil {
		s.writeError(w, r, http.StatusInternalServerError, err)
		return
	}
	if len(session.History) == 0 {
		auditEntries, err := s.storage.ListPassageRepairAudit(r.Context(), opinionID, string(userID))
		if err == nil {
			session.History = historyEntriesToSessionHistory(auditEntries)
		}
	}
	beforeUndo := session.Current
	if err := session.Undo(); err != nil {
		s.writeError(w, r, http.StatusBadRequest, err)
		return
	}
	if err := replacePassages(s.storage, opinionID, session.Current.Passages); err != nil {
		s.writeError(w, r, http.StatusInternalServerError, err)
		return
	}
	target, _ := mvp.NewPassageRepairTarget(opinionID, []mvp.PassageID{session.Current.Passages[0].PassageID})
	undoOperation, _ := mvp.NewAdminPassageOperation(mvp.AdminPassageOperationUndo, target, nil)
	if err := mvp.RecordAuditedPassageRepairOperation(r.Context(), s.storage, string(userID), "browser", undoOperation, beforeUndo, session.Current, time.Now().UTC()); err != nil {
		s.writeError(w, r, http.StatusInternalServerError, err)
		return
	}
	if _, err := mvp.OpenPassage(userID, session.Current.Passages[0].PassageID, s.storage); err != nil {
		s.writeError(w, r, http.StatusInternalServerError, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(repairResponse{
		PassageID: string(session.Current.Passages[0].PassageID),
		Revision:  session.Current.Revision,
	})
}

func (s *server) writeError(w http.ResponseWriter, r *http.Request, status int, err error) {
	if s.logger != nil {
		s.logger.Printf("status=%d method=%s path=%s err=%v", status, r.Method, r.URL.Path, err)
	}
	http.Error(w, err.Error(), status)
}

func (s *server) handleComplete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request completeRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	userID, opinionID, err := s.requestIdentity(request.UserID, request.OpinionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	passageID, err := mvp.NewPassageID(request.PassageID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	progress, err := mvp.CompletePassage(userID, passageID, s.storage, mvp.SystemClock{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if string(progress.OpinionID) != string(opinionID) {
		http.Error(w, "passage does not belong to opinion", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(progressData(progress)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *server) handleQuestion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request questionRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	userID, opinionID, err := s.requestIdentity(request.UserID, request.OpinionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	passageID, err := mvp.NewPassageID(request.PassageID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	passageRecord, err := s.storage.LoadPassage(passageID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	passage, ok := passageRecord.(mvp.Passage)
	if !ok {
		http.Error(w, mvp.ErrWrongRecordType.Error(), http.StatusInternalServerError)
		return
	}
	if passage.OpinionID != opinionID {
		http.Error(w, "passage does not belong to opinion", http.StatusBadRequest)
		return
	}
	opinionRecord, err := s.storage.LoadOpinion(opinionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	opinion, ok := opinionRecord.(mvp.Opinion)
	if !ok {
		http.Error(w, mvp.ErrWrongRecordType.Error(), http.StatusInternalServerError)
		return
	}

	span, err := mvp.SelectSpan(passage, mvp.Offset(request.Start), mvp.Offset(request.End))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	anchor, err := mvp.AnchorSpan(opinionID, passage.SectionID, passage.PassageID, span)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	question, err := mvp.AskQuestion(userID, anchor, mvp.QuestionText(request.Text), mvp.SystemClock{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	question, err = mvp.SaveQuestion(s.storage, question)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	progressRecord, err := s.storage.LoadProgress(userID, opinionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	progress, ok := progressRecord.(mvp.Progress)
	if !ok {
		http.Error(w, mvp.ErrWrongRecordType.Error(), http.StatusInternalServerError)
		return
	}
	if !containsQuestionID(progress.OpenQuestionIDs, question.QuestionID) {
		progress.OpenQuestionIDs = append(progress.OpenQuestionIDs, question.QuestionID)
		progress.UpdatedAt = time.Now().UTC()
		if _, err := s.storage.SaveProgress(progress); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	questions, err := loadQuestions(s.storage, userID, opinionID)
	if err != nil && !errors.Is(err, mvp.ErrNotFound) {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	contextValue, err := mvp.GatherContext(opinion, passage, anchor, questions)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	answer, err := mvp.GuessAnswer(s.model, contextValue, question, mvp.SystemClock{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if _, err := mvp.SaveAnswer(s.storage, answer); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	question.Status = mvp.QuestionStatusAnswered
	if _, err := mvp.SaveQuestion(s.storage, question); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	progress.OpenQuestionIDs = removeQuestionID(progress.OpenQuestionIDs, question.QuestionID)
	progress.UpdatedAt = time.Now().UTC()
	if _, err := s.storage.SaveProgress(progress); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := questionResponse{
		Question: questionItem(question),
		Answer:   answerData(answer),
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *server) requestIdentity(rawUserID, rawOpinionID string) (mvp.UserID, mvp.OpinionID, error) {
	userID := s.userID
	if strings.TrimSpace(rawUserID) != "" {
		parsed, err := mvp.NewUserID(rawUserID)
		if err != nil {
			return "", "", err
		}
		userID = parsed
	}

	opinionID := s.opinionID
	if strings.TrimSpace(rawOpinionID) != "" {
		parsed, err := mvp.NewOpinionID(rawOpinionID)
		if err != nil {
			return "", "", err
		}
		opinionID = parsed
	}
	return userID, opinionID, nil
}

func opinionData(opinion mvp.Opinion) opinionDTO {
	sections := make([]sectionDTO, 0, len(opinion.Sections))
	for _, section := range opinion.Sections {
		sections = append(sections, sectionDTO{
			SectionID: string(section.SectionID),
			Kind:      section.Kind.String(),
			Title:     section.Title,
			StartPage: int(section.StartPage),
			EndPage:   int(section.EndPage),
		})
	}
	return opinionDTO{
		OpinionID: string(opinion.OpinionID),
		CaseName:  opinion.Meta.CaseName,
		Docket:    opinion.Meta.DocketNumber,
		DecidedOn: opinion.Meta.DecidedOn,
		Term:      opinion.Meta.TermLabel,
		Sections:  sections,
	}
}

func passageData(passage mvp.Passage) passageDTO {
	citations := make([]citationDTO, 0, len(passage.Citations))
	for _, citation := range passage.Citations {
		citations = append(citations, citationDTO{
			CitationID: string(citation.CitationID),
			Kind:       citation.Kind.String(),
			RawText:    citation.RawText,
			Normalized: citation.Normalized,
		})
	}
	return passageDTO{
		PassageID:    string(passage.PassageID),
		SectionID:    string(passage.SectionID),
		PageStart:    int(passage.PageStart),
		PageEnd:      int(passage.PageEnd),
		Text:         string(passage.Text),
		FitsOnScreen: passage.FitsOnScreen,
		Citations:    citations,
	}
}

func progressData(progress mvp.Progress) progressDTO {
	completed := make([]string, 0, len(progress.CompletedPassages))
	for _, passageID := range progress.CompletedPassages {
		completed = append(completed, string(passageID))
	}
	questions := make([]string, 0, len(progress.OpenQuestionIDs))
	for _, questionID := range progress.OpenQuestionIDs {
		questions = append(questions, string(questionID))
	}

	currentPassageID := ""
	if progress.CurrentPassage != nil {
		currentPassageID = string(*progress.CurrentPassage)
	}

	return progressDTO{
		UserID:            string(progress.UserID),
		OpinionID:         string(progress.OpinionID),
		CurrentPassageID:  currentPassageID,
		CompletedPassages: completed,
		OpenQuestionIDs:   questions,
	}
}

func passageListData(passages []mvp.Passage) []passageListItem {
	items := make([]passageListItem, 0, len(passages))
	for index, passage := range passages {
		items = append(items, passageListItem{
			PassageID: string(passage.PassageID),
			SectionID: string(passage.SectionID),
			Label:     fmt.Sprintf("Passage %d", index+1),
		})
	}
	return items
}

func questionData(questions []mvp.Question) []questionDTO {
	items := make([]questionDTO, 0, len(questions))
	for _, question := range questions {
		items = append(items, questionItem(question))
	}
	return items
}

func questionItem(question mvp.Question) questionDTO {
	return questionDTO{
		QuestionID: string(question.QuestionID),
		Text:       string(question.Text),
		Status:     question.Status.String(),
		PassageID:  string(question.Anchor.PassageID),
		Quote:      string(question.Anchor.Span.Quote),
	}
}

func answerData(answer mvp.AnswerDraft) answerDTO {
	evidence := make([]evidenceDTO, 0, len(answer.Evidence))
	for _, item := range answer.Evidence {
		evidence = append(evidence, evidenceDTO{
			Label:     item.Label,
			Quote:     string(item.Quote),
			PassageID: string(item.Anchor.PassageID),
		})
	}
	return answerDTO{
		QuestionID: string(answer.QuestionID),
		Answer:     string(answer.Answer),
		Evidence:   evidence,
		Caveats:    append([]string(nil), answer.Caveats...),
		ModelName:  answer.ModelName,
	}
}

func repairData(storage *mvp.SQLiteStorage, opinionID mvp.OpinionID, userID mvp.UserID, passages []mvp.Passage, focus mvp.Passage) repairDTO {
	index := 0
	for i, passage := range passages {
		if passage.PassageID == focus.PassageID {
			index = i
			break
		}
	}
	var previous *mvp.Passage
	var next *mvp.Passage
	if index > 0 {
		previous = &passages[index-1]
	}
	if index+1 < len(passages) {
		next = &passages[index+1]
	}
	issues, _ := mvp.ClassifyPassageIssues(focus, previous, next)
	var entries []mvp.PassageRepairAuditEntry
	if storage != nil {
		entries, _ = storage.ListPassageRepairAudit(context.Background(), opinionID, string(userID))
	}
	history := make([]repairHistoryDTO, 0, len(entries))
	for _, entry := range filterRepairAuditForPassage(entries, focus.PassageID) {
		history = append(history, repairHistoryDTO{
			Revision:      entry.Revision,
			OperationKind: string(entry.OperationKind),
			TargetPassage: string(entry.TargetPassageID),
			CreatedAt:     entry.CreatedAt.UTC().Format(time.RFC3339Nano),
		})
	}
	issueData := make([]repairIssueDTO, 0, len(issues))
	for _, issue := range issues {
		issueData = append(issueData, repairIssueDTO{
			Kind:    string(issue.Kind),
			Summary: issue.Summary,
		})
	}
	return repairDTO{
		Issues:           issueData,
		History:          history,
		CanMergeNext:     next != nil,
		CanMergePrevious: previous != nil,
		CanSplitSentence: focus.SentenceStart < focus.SentenceEnd,
		CanRemoveHeader:  hasRepairIssue(issues, mvp.PassageIssuePageHeaderArtifact),
	}
}

func filterRepairAuditForPassage(entries []mvp.PassageRepairAuditEntry, focusPassageID mvp.PassageID) []mvp.PassageRepairAuditEntry {
	filtered := make([]mvp.PassageRepairAuditEntry, 0, len(entries))
	for _, entry := range entries {
		if repairEntryTouchesPassage(entry, focusPassageID) {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

func repairEntryTouchesPassage(entry mvp.PassageRepairAuditEntry, focusPassageID mvp.PassageID) bool {
	if entry.TargetPassageID == focusPassageID {
		return true
	}

	switch entry.OperationKind {
	case mvp.AdminPassageOperationMergeWithPrevious, mvp.AdminPassageOperationMoveFirstSentencePrev:
		previousID, ok := previousPassageID(entry.Before.Passages, entry.TargetPassageID)
		return ok && previousID == focusPassageID
	case mvp.AdminPassageOperationMergeWithNext, mvp.AdminPassageOperationMoveLastSentenceNext:
		nextID, ok := nextPassageID(entry.Before.Passages, entry.TargetPassageID)
		return ok && nextID == focusPassageID
	default:
		return false
	}
}

func previousPassageID(passages []mvp.Passage, target mvp.PassageID) (mvp.PassageID, bool) {
	for index, passage := range passages {
		if passage.PassageID == target && index > 0 {
			return passages[index-1].PassageID, true
		}
	}
	return "", false
}

func nextPassageID(passages []mvp.Passage, target mvp.PassageID) (mvp.PassageID, bool) {
	for index, passage := range passages {
		if passage.PassageID == target && index+1 < len(passages) {
			return passages[index+1].PassageID, true
		}
	}
	return "", false
}

func parseBrowserRepairOperation(action string, storage *mvp.SQLiteStorage, opinionID mvp.OpinionID, passageID mvp.PassageID) (mvp.AdminPassageOperation, error) {
	target, err := mvp.NewPassageRepairTarget(opinionID, []mvp.PassageID{passageID})
	if err != nil {
		return mvp.AdminPassageOperation{}, err
	}
	switch strings.TrimSpace(action) {
	case "mergeNext":
		return mvp.NewAdminPassageOperation(mvp.AdminPassageOperationMergeWithNext, target, nil)
	case "mergePrevious":
		return mvp.NewAdminPassageOperation(mvp.AdminPassageOperationMergeWithPrevious, target, nil)
	case "splitSentence":
		record, err := storage.LoadPassage(passageID)
		if err != nil {
			return mvp.AdminPassageOperation{}, err
		}
		passage := record.(mvp.Passage)
		splitAfter := passage.SentenceStart
		return mvp.NewAdminPassageOperation(mvp.AdminPassageOperationSplitAtSentence, target, &splitAfter)
	case "removeHeader":
		return mvp.NewAdminPassageOperation(mvp.AdminPassageOperationRemoveRunningHeader, target, nil)
	default:
		return mvp.AdminPassageOperation{}, errors.New("unknown repair action")
	}
}

func browserRepairFocusPassageID(operation mvp.AdminPassageOperation, before mvp.PassageRepairSnapshot, after mvp.PassageRepairSnapshot) mvp.PassageID {
	targetID := operation.Target.PassageIDs[0]
	for _, passage := range after.Passages {
		if passage.PassageID == targetID {
			return passage.PassageID
		}
	}

	if operation.Kind == mvp.AdminPassageOperationMergeWithPrevious {
		for index, passage := range before.Passages {
			if passage.PassageID == targetID && index > 0 {
				previousID := before.Passages[index-1].PassageID
				for _, merged := range after.Passages {
					if merged.PassageID == previousID {
						return previousID
					}
				}
			}
		}
	}

	if len(after.Passages) > 0 {
		return after.Passages[0].PassageID
	}
	return targetID
}

func historyEntriesToSessionHistory(entries []mvp.PassageRepairAuditEntry) []mvp.PassageRepairHistoryEntry {
	out := make([]mvp.PassageRepairHistoryEntry, 0, len(entries))
	for _, entry := range entries {
		target, _ := mvp.NewPassageRepairTarget(entry.OpinionID, []mvp.PassageID{entry.TargetPassageID})
		operation, _ := mvp.NewAdminPassageOperation(entry.OperationKind, target, entry.SplitAfterSentence)
		out = append(out, mvp.PassageRepairHistoryEntry{
			Revision:  entry.Revision,
			Operation: operation,
			Before:    entry.Before,
			After:     entry.After,
		})
	}
	return out
}

func hasRepairIssue(issues []mvp.PassageIssue, kind mvp.PassageIssueKind) bool {
	for _, issue := range issues {
		if issue.Kind == kind {
			return true
		}
	}
	return false
}

func loadQuestions(storage *mvp.SQLiteStorage, userID mvp.UserID, opinionID mvp.OpinionID) ([]mvp.Question, error) {
	records, err := storage.LoadQuestions(userID, opinionID)
	if err != nil {
		return nil, err
	}
	questions := make([]mvp.Question, 0, len(records))
	for _, record := range records {
		question, ok := record.(mvp.Question)
		if !ok {
			return nil, mvp.ErrWrongRecordType
		}
		questions = append(questions, question)
	}
	return questions, nil
}

func listPassages(storage *mvp.SQLiteStorage, opinionID mvp.OpinionID) ([]mvp.Passage, error) {
	records, err := storage.ListPassages(opinionID)
	if err != nil {
		return nil, err
	}
	passages := make([]mvp.Passage, 0, len(records))
	for _, record := range records {
		passage, ok := record.(mvp.Passage)
		if !ok {
			return nil, mvp.ErrWrongRecordType
		}
		passages = append(passages, passage)
	}
	sort.Slice(passages, func(i, j int) bool {
		if passages[i].PageStart != passages[j].PageStart {
			return passages[i].PageStart < passages[j].PageStart
		}
		if passages[i].SentenceStart != passages[j].SentenceStart {
			return passages[i].SentenceStart < passages[j].SentenceStart
		}
		return passages[i].PassageID < passages[j].PassageID
	})
	return passages, nil
}

func replacePassages(storage *mvp.SQLiteStorage, opinionID mvp.OpinionID, passages []mvp.Passage) error {
	rewriter, ok := any(storage).(mvp.PassageRewriter)
	if !ok {
		return mvp.ErrWrongRecordType
	}
	records := make([]mvp.PassageRecord, 0, len(passages))
	for _, passage := range passages {
		records = append(records, passage)
	}
	return rewriter.ReplacePassages(opinionID, records)
}

func containsQuestionID(ids []mvp.QuestionID, candidate mvp.QuestionID) bool {
	for _, id := range ids {
		if id == candidate {
			return true
		}
	}
	return false
}

func removeQuestionID(ids []mvp.QuestionID, candidate mvp.QuestionID) []mvp.QuestionID {
	filtered := make([]mvp.QuestionID, 0, len(ids))
	for _, id := range ids {
		if id == candidate {
			continue
		}
		filtered = append(filtered, id)
	}
	return filtered
}
