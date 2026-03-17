package mvp

import (
	"bytes"
	"context"
	"database/sql"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestApplyMigrationsCreatesTables(t *testing.T) {
	clock := FixedClock{Time: time.Date(2026, time.March, 17, 0, 15, 0, 0, time.UTC)}
	storage, path, err := NewTempSQLite(clock)
	if err != nil {
		t.Fatalf("new temp sqlite: %v", err)
	}
	defer os.RemoveAll(filepath.Dir(path))
	defer storage.Close()

	tables := []string{
		"schema_migrations",
		"raw_pdfs",
		"opinions",
		"sections",
		"passages",
		"citations",
		"progress",
		"completed_passages",
		"questions",
	}

	for _, table := range tables {
		var count int
		err := storage.db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&count)
		if err != nil {
			t.Fatalf("query sqlite_master for %s: %v", table, err)
		}
		if count != 1 {
			t.Fatalf("expected table %s to exist", table)
		}
	}
}

func TestApplyMigrationsIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	db, err := sql.Open("sqlite3", filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	now := time.Date(2026, time.March, 17, 0, 20, 0, 0, time.UTC)
	if err := ApplyMigrations(context.Background(), db, DefaultMigrations(), now); err != nil {
		t.Fatalf("first migration apply: %v", err)
	}
	if err := ApplyMigrations(context.Background(), db, DefaultMigrations(), now.Add(time.Minute)); err != nil {
		t.Fatalf("second migration apply: %v", err)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&count); err != nil {
		t.Fatalf("count schema_migrations: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 applied migration, got %d", count)
	}
}

func TestSQLiteRawPDFAndOpinionRoundTrip(t *testing.T) {
	storage, cleanup := newSQLiteFixtureStorage(t)
	defer cleanup()

	opinion := sampleOpinionFixture(t)
	raw := sampleRawPDFFixture(t, opinion.OpinionID)

	if _, err := saveRawPDF(storage, raw); err != nil {
		t.Fatalf("save raw pdf: %v", err)
	}
	if _, err := saveOpinion(storage, opinion); err != nil {
		t.Fatalf("save opinion: %v", err)
	}

	gotRaw, err := loadRawPDF(storage, raw.OpinionID)
	if err != nil {
		t.Fatalf("load raw pdf: %v", err)
	}
	if gotRaw.SHA256 != raw.SHA256 {
		t.Fatalf("got raw hash %q, want %q", gotRaw.SHA256, raw.SHA256)
	}

	gotOpinion, err := loadOpinion(storage, opinion.OpinionID)
	if err != nil {
		t.Fatalf("load opinion: %v", err)
	}
	if len(gotOpinion.Sections) != len(opinion.Sections) {
		t.Fatalf("got %d sections, want %d", len(gotOpinion.Sections), len(opinion.Sections))
	}
}

func TestSQLitePassageProgressAndQuestionRoundTrip(t *testing.T) {
	storage, cleanup := newSQLiteFixtureStorage(t)
	defer cleanup()

	opinion := sampleOpinionFixture(t)
	if _, err := saveOpinion(storage, opinion); err != nil {
		t.Fatalf("save opinion: %v", err)
	}

	passages := samplePassagesFixture(t, opinion.OpinionID, opinion.Sections[0].SectionID)
	if _, err := savePassages(storage, passages); err != nil {
		t.Fatalf("save passages: %v", err)
	}

	userID, _ := NewUserID("reader-1")
	current := passages[0].PassageID
	progress, _ := NewProgress(userID, opinion.OpinionID, &current, []PassageID{current}, nil, time.Now())
	if _, err := saveProgress(storage, progress); err != nil {
		t.Fatalf("save progress: %v", err)
	}

	span, _ := NewSpan(0, 8, "Passage 1")
	anchor, _ := NewAnchor(opinion.OpinionID, opinion.Sections[0].SectionID, current, span)
	questionID, _ := NewQuestionID("q-1")
	question, _ := NewQuestion(questionID, userID, anchor, "What is the standard?", time.Now(), QuestionStatusOpen)
	if _, err := saveQuestion(storage, question); err != nil {
		t.Fatalf("save question: %v", err)
	}

	gotPassage, err := loadPassage(storage, current)
	if err != nil {
		t.Fatalf("load passage: %v", err)
	}
	if len(gotPassage.Citations) == 0 {
		t.Fatalf("expected saved passage citations")
	}

	gotProgress, err := loadProgress(storage, userID, opinion.OpinionID)
	if err != nil {
		t.Fatalf("load progress: %v", err)
	}
	if gotProgress.CurrentPassage == nil || *gotProgress.CurrentPassage != current {
		t.Fatalf("unexpected current passage: %+v", gotProgress.CurrentPassage)
	}

	questions, err := loadQuestions(storage, userID, opinion.OpinionID)
	if err != nil {
		t.Fatalf("load questions: %v", err)
	}
	if len(questions) != 1 {
		t.Fatalf("got %d questions, want 1", len(questions))
	}
}

func TestRunMVPRollsBackSQLiteTransactionOnGuessFailure(t *testing.T) {
	storage, cleanup := newSQLiteFixtureStorage(t)
	defer cleanup()
	fixture := loadRealFixturePDF(t)

	userID, _ := NewUserID("reader-1")
	input, _ := NewUserInput(URL("https://www.supremecourt.gov/opinions/25pdf/24-777_9ol1.pdf"))
	client := testHTTPClient(fixture.Bytes)
	clock := FixedClock{Time: time.Date(2026, time.March, 17, 0, 30, 0, 0, time.UTC)}

	err := runMVPExpectError(context.Background(), userID, input, client, storage, clock, Model{})
	if err == nil {
		t.Fatalf("expected run mvp to fail with invalid model")
	}

	assertTableEmpty(t, storage.db, "raw_pdfs")
	assertTableEmpty(t, storage.db, "opinions")
	assertTableEmpty(t, storage.db, "passages")
}

func TestRunMVPAgainstMemoryAndSQLite(t *testing.T) {
	fixture := loadRealFixturePDF(t)
	userID, _ := NewUserID("reader-1")
	input, _ := NewUserInput(URL("https://www.supremecourt.gov/opinions/25pdf/24-777_9ol1.pdf"))
	client := testHTTPClient(fixture.Bytes)
	clock := FixedClock{Time: time.Date(2026, time.March, 17, 0, 35, 0, 0, time.UTC)}
	model, _ := NewModel("gpt-5", 8000)

	memory := NewMemoryStorage()
	memoryState, err := RunMVP(context.Background(), userID, input, client, memory, clock, model)
	if err != nil {
		t.Fatalf("run mvp with memory: %v", err)
	}

	sqliteStorage, cleanup := newSQLiteFixtureStorage(t)
	defer cleanup()
	sqliteState, err := RunMVP(context.Background(), userID, input, client, sqliteStorage, clock, model)
	if err != nil {
		t.Fatalf("run mvp with sqlite: %v", err)
	}

	if memoryState.Passage.PassageID == "" || sqliteState.Passage.PassageID == "" {
		t.Fatalf("expected both backends to produce a reading state")
	}
}

func TestSQLiteBrowserConstraintsProof(t *testing.T) {
	bytes, err := os.ReadFile(filepath.Join("..", "sqlite_browser.md"))
	if err != nil {
		t.Fatalf("read sqlite browser note: %v", err)
	}
	text := string(bytes)
	if !strings.Contains(text, "browser-hosted SQLite") {
		t.Fatalf("expected browser-hosted SQLite note")
	}
	if strings.Contains(strings.ToLower(schema0001), "virtual table") {
		t.Fatalf("schema should avoid virtual tables for browser portability")
	}
	if strings.Contains(strings.ToLower(schema0001), "load_extension") {
		t.Fatalf("schema should not rely on loadable extensions")
	}
}

func newSQLiteFixtureStorage(t *testing.T) (*SQLiteStorage, func()) {
	t.Helper()
	clock := FixedClock{Time: time.Date(2026, time.March, 17, 0, 10, 0, 0, time.UTC)}
	storage, path, err := NewTempSQLite(clock)
	if err != nil {
		t.Fatalf("new temp sqlite: %v", err)
	}
	return storage, func() {
		_ = storage.Close()
		_ = os.RemoveAll(filepath.Dir(path))
	}
}

func sampleRawPDFFixture(t *testing.T, opinionID OpinionID) RawPDF {
	t.Helper()
	raw, err := NewRawPDF(opinionID, URL("https://example.com/opinion.pdf"), PDFBytes("sample pdf bytes"), time.Now())
	if err != nil {
		t.Fatalf("new raw pdf: %v", err)
	}
	return raw
}

func sampleOpinionFixture(t *testing.T) Opinion {
	t.Helper()
	opinionID, _ := NewOpinionID("24-777")
	sectionID, _ := NewSectionID("syllabus")
	meta, _ := NewMeta("Urias-Orellana v. Bondi", "24-777", "March 4, 2026", "2025", nil)
	section, err := NewSection(opinionID, sectionID, SectionKindSyllabus, "Syllabus", nil, 1, 2, "Syllabus text")
	if err != nil {
		t.Fatalf("new section: %v", err)
	}
	opinion, err := NewOpinion(opinionID, meta, []Section{section}, "Syllabus text")
	if err != nil {
		t.Fatalf("new opinion: %v", err)
	}
	return opinion
}

func samplePassagesFixture(t *testing.T, opinionID OpinionID, sectionID SectionID) []Passage {
	t.Helper()
	passageID, _ := NewPassageID("p-1")
	span, _ := NewSpan(0, 13, "Roe v. Wade")
	citationID, _ := NewCitationID("c-1")
	citation, _ := NewCitation(citationID, CitationKindCase, "Roe v. Wade", nil, span)
	passage, err := NewPassage(passageID, opinionID, sectionID, 0, 0, 1, 1, "Passage 1 Roe v. Wade.", []Citation{citation}, true)
	if err != nil {
		t.Fatalf("new passage: %v", err)
	}
	return []Passage{passage}
}

func testHTTPClient(body []byte) *http.Client {
	return &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader(body)),
				Header:     make(http.Header),
			}, nil
		}),
	}
}

func runMVPExpectError(ctx context.Context, userID UserID, input UserInput, client *http.Client, storage Storage, clock Clock, model Model) error {
	_, err := RunMVP(ctx, userID, input, client, storage, clock, model)
	return err
}

func assertTableEmpty(t *testing.T, db *sql.DB, table string) {
	t.Helper()
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&count); err != nil {
		t.Fatalf("count rows in %s: %v", table, err)
	}
	if count != 0 {
		t.Fatalf("expected table %s to be empty, got %d rows", table, count)
	}
}
