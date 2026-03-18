package mvp

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed sql/schema/0001_init.sql
var schema0001 string

//go:embed sql/schema/0002_repair_audit.sql
var schema0002 string

//go:embed sql/schema/0003_answers.sql
var schema0003 string

var (
	ErrMigrationVersionMissing = errors.New("migration version is required")
)

type TransactionalStorage interface {
	InTx(func(Storage) error) error
}

type SQLiteStorage struct {
	db *sql.DB
}

type sqliteExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type sqliteTxStorage struct {
	parent *SQLiteStorage
	tx     *sql.Tx
}

type Migration struct {
	Version string
	SQL     string
}

func OpenSQLite(path string) (*SQLiteStorage, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`PRAGMA foreign_keys = ON;`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}
	return &SQLiteStorage{db: db}, nil
}

func (s *SQLiteStorage) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func DefaultMigrations() []Migration {
	return []Migration{
		{Version: "0001_init", SQL: schema0001},
		{Version: "0002_repair_audit", SQL: schema0002},
		{Version: "0003_answers", SQL: schema0003},
	}
}

func ApplyMigrations(ctx context.Context, db *sql.DB, migrations []Migration, now Timestamp) error {
	if len(migrations) == 0 {
		return nil
	}
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	if _, err := db.ExecContext(ctx, `PRAGMA foreign_keys = ON;`); err != nil {
		return fmt.Errorf("enable foreign keys: %w", err)
	}
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT PRIMARY KEY, applied_at TEXT NOT NULL);`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	for _, migration := range migrations {
		if strings.TrimSpace(migration.Version) == "" {
			return ErrMigrationVersionMissing
		}
		var appliedAt string
		err := db.QueryRowContext(ctx, `SELECT applied_at FROM schema_migrations WHERE version = ?`, migration.Version).Scan(&appliedAt)
		if err == nil {
			continue
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("check migration %s: %w", migration.Version, err)
		}

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", migration.Version, err)
		}
		if _, err := tx.ExecContext(ctx, migration.SQL); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply migration %s: %w", migration.Version, err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations(version, applied_at) VALUES(?, ?)`, migration.Version, now.UTC().Format(time.RFC3339Nano)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration %s: %w", migration.Version, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", migration.Version, err)
		}
	}
	return nil
}

func OpenSQLiteWithMigrations(path string, clock Clock) (*SQLiteStorage, error) {
	storage, err := OpenSQLite(path)
	if err != nil {
		return nil, err
	}
	if err := ApplyMigrations(context.Background(), storage.db, DefaultMigrations(), clock.Now()); err != nil {
		_ = storage.Close()
		return nil, err
	}
	return storage, nil
}

func (s *SQLiteStorage) InTx(fn func(Storage) error) error {
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	txStorage := &sqliteTxStorage{parent: s, tx: tx}
	if err := fn(txStorage); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func NewTempSQLite(clock Clock) (*SQLiteStorage, string, error) {
	dir, err := os.MkdirTemp("", "sprout-sqlite-*")
	if err != nil {
		return nil, "", err
	}
	path := filepath.Join(dir, "sprout.db")
	storage, err := OpenSQLiteWithMigrations(path, clock)
	if err != nil {
		return nil, "", err
	}
	return storage, path, nil
}

func (s *SQLiteStorage) SaveRawPDF(record RawPDFRecord) (RawPDFRecord, error) {
	raw, err := rawPDFFromRecord(record)
	if err != nil {
		return nil, err
	}
	if err := saveRawPDFExec(context.Background(), s.db, raw); err != nil {
		return nil, err
	}
	return raw, nil
}

func (s *SQLiteStorage) SaveOpinion(record OpinionRecord) (OpinionRecord, error) {
	opinion, err := opinionFromRecord(record)
	if err != nil {
		return nil, err
	}
	if err := saveOpinionExec(context.Background(), s.db, opinion); err != nil {
		return nil, err
	}
	return opinion, nil
}

func (s *SQLiteStorage) SavePassages(records []PassageRecord) ([]PassageRecord, error) {
	passages := make([]Passage, 0, len(records))
	for _, record := range records {
		passage, err := passageFromRecord(record)
		if err != nil {
			return nil, err
		}
		passages = append(passages, passage)
	}
	if err := savePassagesExec(context.Background(), s.db, passages); err != nil {
		return nil, err
	}
	out := make([]PassageRecord, 0, len(passages))
	for _, passage := range passages {
		out = append(out, passage)
	}
	return out, nil
}

func (s *SQLiteStorage) SaveProgress(record ProgressRecord) (ProgressRecord, error) {
	progress, err := progressFromRecord(record)
	if err != nil {
		return nil, err
	}
	if err := saveProgressExec(context.Background(), s.db, progress); err != nil {
		return nil, err
	}
	return progress, nil
}

func (s *SQLiteStorage) SaveQuestionRecord(record QuestionRecord) (QuestionRecord, error) {
	question, err := questionFromRecord(record)
	if err != nil {
		return nil, err
	}
	if err := saveQuestionExec(context.Background(), s.db, question); err != nil {
		return nil, err
	}
	return question, nil
}

func (s *SQLiteStorage) SaveAnswerRecord(record AnswerRecord) (AnswerRecord, error) {
	answer, err := answerFromRecord(record)
	if err != nil {
		return nil, err
	}
	if err := saveAnswerExec(context.Background(), s.db, answer); err != nil {
		return nil, err
	}
	return answer, nil
}

func (s *SQLiteStorage) LoadRawPDF(opinionID OpinionID) (RawPDFRecord, error) {
	return loadRawPDFExec(context.Background(), s.db, opinionID)
}

func (s *SQLiteStorage) LoadOpinion(opinionID OpinionID) (OpinionRecord, error) {
	return loadOpinionExec(context.Background(), s.db, opinionID)
}

func (s *SQLiteStorage) LoadPassage(passageID PassageID) (PassageRecord, error) {
	return loadPassageExec(context.Background(), s.db, passageID)
}

func (s *SQLiteStorage) LoadProgress(userID UserID, opinionID OpinionID) (ProgressRecord, error) {
	return loadProgressExec(context.Background(), s.db, userID, opinionID)
}

func (s *SQLiteStorage) LoadQuestions(userID UserID, opinionID OpinionID) ([]QuestionRecord, error) {
	questions, err := loadQuestionsExec(context.Background(), s.db, userID, opinionID)
	if err != nil {
		return nil, err
	}
	out := make([]QuestionRecord, 0, len(questions))
	for _, question := range questions {
		out = append(out, question)
	}
	return out, nil
}

func (s *SQLiteStorage) LoadAnswers(userID UserID, opinionID OpinionID) ([]AnswerRecord, error) {
	answers, err := loadAnswersExec(context.Background(), s.db, userID, opinionID)
	if err != nil {
		return nil, err
	}
	out := make([]AnswerRecord, 0, len(answers))
	for _, answer := range answers {
		out = append(out, answer)
	}
	return out, nil
}

func (s *SQLiteStorage) ListPassages(opinionID OpinionID) ([]PassageRecord, error) {
	passages, err := listPassagesExec(context.Background(), s.db, opinionID)
	if err != nil {
		return nil, err
	}
	out := make([]PassageRecord, 0, len(passages))
	for _, passage := range passages {
		out = append(out, passage)
	}
	return out, nil
}

func (s *SQLiteStorage) ReplacePassages(opinionID OpinionID, records []PassageRecord) error {
	passages := make([]Passage, 0, len(records))
	for _, record := range records {
		passage, err := passageFromRecord(record)
		if err != nil {
			return err
		}
		passages = append(passages, passage)
	}
	return replacePassagesExec(context.Background(), s.db, opinionID, passages)
}

func (s *sqliteTxStorage) SaveRawPDF(record RawPDFRecord) (RawPDFRecord, error) {
	raw, err := rawPDFFromRecord(record)
	if err != nil {
		return nil, err
	}
	if err := saveRawPDFExec(context.Background(), s.tx, raw); err != nil {
		return nil, err
	}
	return raw, nil
}

func (s *sqliteTxStorage) SaveOpinion(record OpinionRecord) (OpinionRecord, error) {
	opinion, err := opinionFromRecord(record)
	if err != nil {
		return nil, err
	}
	if err := saveOpinionExec(context.Background(), s.tx, opinion); err != nil {
		return nil, err
	}
	return opinion, nil
}

func (s *sqliteTxStorage) SavePassages(records []PassageRecord) ([]PassageRecord, error) {
	passages := make([]Passage, 0, len(records))
	for _, record := range records {
		passage, err := passageFromRecord(record)
		if err != nil {
			return nil, err
		}
		passages = append(passages, passage)
	}
	if err := savePassagesExec(context.Background(), s.tx, passages); err != nil {
		return nil, err
	}
	out := make([]PassageRecord, 0, len(passages))
	for _, passage := range passages {
		out = append(out, passage)
	}
	return out, nil
}

func (s *sqliteTxStorage) SaveProgress(record ProgressRecord) (ProgressRecord, error) {
	progress, err := progressFromRecord(record)
	if err != nil {
		return nil, err
	}
	if err := saveProgressExec(context.Background(), s.tx, progress); err != nil {
		return nil, err
	}
	return progress, nil
}

func (s *sqliteTxStorage) SaveQuestionRecord(record QuestionRecord) (QuestionRecord, error) {
	question, err := questionFromRecord(record)
	if err != nil {
		return nil, err
	}
	if err := saveQuestionExec(context.Background(), s.tx, question); err != nil {
		return nil, err
	}
	return question, nil
}

func (s *sqliteTxStorage) SaveAnswerRecord(record AnswerRecord) (AnswerRecord, error) {
	answer, err := answerFromRecord(record)
	if err != nil {
		return nil, err
	}
	if err := saveAnswerExec(context.Background(), s.tx, answer); err != nil {
		return nil, err
	}
	return answer, nil
}

func (s *sqliteTxStorage) LoadRawPDF(opinionID OpinionID) (RawPDFRecord, error) {
	return loadRawPDFExec(context.Background(), s.tx, opinionID)
}

func (s *sqliteTxStorage) LoadOpinion(opinionID OpinionID) (OpinionRecord, error) {
	return loadOpinionExec(context.Background(), s.tx, opinionID)
}

func (s *sqliteTxStorage) LoadPassage(passageID PassageID) (PassageRecord, error) {
	return loadPassageExec(context.Background(), s.tx, passageID)
}

func (s *sqliteTxStorage) LoadProgress(userID UserID, opinionID OpinionID) (ProgressRecord, error) {
	return loadProgressExec(context.Background(), s.tx, userID, opinionID)
}

func (s *sqliteTxStorage) LoadQuestions(userID UserID, opinionID OpinionID) ([]QuestionRecord, error) {
	questions, err := loadQuestionsExec(context.Background(), s.tx, userID, opinionID)
	if err != nil {
		return nil, err
	}
	out := make([]QuestionRecord, 0, len(questions))
	for _, question := range questions {
		out = append(out, question)
	}
	return out, nil
}

func (s *sqliteTxStorage) LoadAnswers(userID UserID, opinionID OpinionID) ([]AnswerRecord, error) {
	answers, err := loadAnswersExec(context.Background(), s.tx, userID, opinionID)
	if err != nil {
		return nil, err
	}
	out := make([]AnswerRecord, 0, len(answers))
	for _, answer := range answers {
		out = append(out, answer)
	}
	return out, nil
}

func (s *sqliteTxStorage) ListPassages(opinionID OpinionID) ([]PassageRecord, error) {
	passages, err := listPassagesExec(context.Background(), s.tx, opinionID)
	if err != nil {
		return nil, err
	}
	out := make([]PassageRecord, 0, len(passages))
	for _, passage := range passages {
		out = append(out, passage)
	}
	return out, nil
}

func (s *sqliteTxStorage) ReplacePassages(opinionID OpinionID, records []PassageRecord) error {
	passages := make([]Passage, 0, len(records))
	for _, record := range records {
		passage, err := passageFromRecord(record)
		if err != nil {
			return err
		}
		passages = append(passages, passage)
	}
	return replacePassagesExec(context.Background(), s.tx, opinionID, passages)
}

func saveRawPDFExec(ctx context.Context, exec sqliteExecutor, raw RawPDF) error {
	_, err := exec.ExecContext(ctx, `
		INSERT INTO raw_pdfs(opinion_id, source_url, bytes, fetched_at, sha256)
		VALUES(?, ?, ?, ?, ?)
		ON CONFLICT(opinion_id) DO UPDATE SET
			source_url = excluded.source_url,
			bytes = excluded.bytes,
			fetched_at = excluded.fetched_at,
			sha256 = excluded.sha256
	`, raw.OpinionID, raw.SourceURL, raw.Bytes, raw.FetchedAt.UTC().Format(time.RFC3339Nano), raw.SHA256)
	return err
}

func saveOpinionExec(ctx context.Context, exec sqliteExecutor, opinion Opinion) error {
	if _, err := exec.ExecContext(ctx, `
		INSERT INTO opinions(opinion_id, case_name, docket_number, decided_on, term_label, primary_author, full_text)
		VALUES(?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(opinion_id) DO UPDATE SET
			case_name = excluded.case_name,
			docket_number = excluded.docket_number,
			decided_on = excluded.decided_on,
			term_label = excluded.term_label,
			primary_author = excluded.primary_author,
			full_text = excluded.full_text
	`, opinion.OpinionID, opinion.Meta.CaseName, opinion.Meta.DocketNumber, opinion.Meta.DecidedOn, opinion.Meta.TermLabel, derefJustice(opinion.Meta.PrimaryAuthor), opinion.FullText); err != nil {
		return err
	}
	if _, err := exec.ExecContext(ctx, `DELETE FROM sections WHERE opinion_id = ?`, opinion.OpinionID); err != nil {
		return err
	}
	for index, section := range opinion.Sections {
		if _, err := exec.ExecContext(ctx, `
			INSERT INTO sections(opinion_id, section_id, kind, title, author, start_page, end_page, text, sort_index)
			VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, section.OpinionID, section.SectionID, section.Kind.String(), section.Title, derefJustice(section.Author), section.StartPage, section.EndPage, section.Text, index); err != nil {
			return err
		}
	}
	return nil
}

func savePassagesExec(ctx context.Context, exec sqliteExecutor, passages []Passage) error {
	for _, passage := range passages {
		if _, err := exec.ExecContext(ctx, `
			INSERT INTO passages(passage_id, opinion_id, section_id, sentence_start, sentence_end, page_start, page_end, text, fits_on_screen)
			VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(passage_id) DO UPDATE SET
				opinion_id = excluded.opinion_id,
				section_id = excluded.section_id,
				sentence_start = excluded.sentence_start,
				sentence_end = excluded.sentence_end,
				page_start = excluded.page_start,
				page_end = excluded.page_end,
				text = excluded.text,
				fits_on_screen = excluded.fits_on_screen
		`, passage.PassageID, passage.OpinionID, passage.SectionID, passage.SentenceStart, passage.SentenceEnd, passage.PageStart, passage.PageEnd, passage.Text, boolToInt(passage.FitsOnScreen)); err != nil {
			return err
		}
		if _, err := exec.ExecContext(ctx, `DELETE FROM citations WHERE passage_id = ?`, passage.PassageID); err != nil {
			return err
		}
		for index, citation := range passage.Citations {
			if _, err := exec.ExecContext(ctx, `
				INSERT INTO citations(citation_id, passage_id, kind, raw_text, normalized, start_offset, end_offset, quote, sort_index)
				VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, citation.CitationID, passage.PassageID, citation.Kind.String(), citation.RawText, derefString(citation.Normalized), citation.Span.StartOffset, citation.Span.EndOffset, citation.Span.Quote, index); err != nil {
				return err
			}
		}
	}
	return nil
}

func replacePassagesExec(ctx context.Context, exec sqliteExecutor, opinionID OpinionID, passages []Passage) error {
	if _, err := exec.ExecContext(ctx, `DELETE FROM passages WHERE opinion_id = ?`, opinionID); err != nil {
		return err
	}
	return savePassagesExec(ctx, exec, passages)
}

func saveProgressExec(ctx context.Context, exec sqliteExecutor, progress Progress) error {
	if _, err := exec.ExecContext(ctx, `
		INSERT INTO progress(user_id, opinion_id, current_passage_id, updated_at)
		VALUES(?, ?, ?, ?)
		ON CONFLICT(user_id, opinion_id) DO UPDATE SET
			current_passage_id = excluded.current_passage_id,
			updated_at = excluded.updated_at
	`, progress.UserID, progress.OpinionID, derefPassageID(progress.CurrentPassage), progress.UpdatedAt.UTC().Format(time.RFC3339Nano)); err != nil {
		return err
	}
	if _, err := exec.ExecContext(ctx, `DELETE FROM completed_passages WHERE user_id = ? AND opinion_id = ?`, progress.UserID, progress.OpinionID); err != nil {
		return err
	}
	for index, passageID := range progress.CompletedPassages {
		if _, err := exec.ExecContext(ctx, `
			INSERT INTO completed_passages(user_id, opinion_id, passage_id, sort_index)
			VALUES(?, ?, ?, ?)
		`, progress.UserID, progress.OpinionID, passageID, index); err != nil {
			return err
		}
	}
	return nil
}

func saveQuestionExec(ctx context.Context, exec sqliteExecutor, question Question) error {
	_, err := exec.ExecContext(ctx, `
		INSERT INTO questions(question_id, user_id, opinion_id, section_id, passage_id, start_offset, end_offset, quote, text, asked_at, status)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(question_id) DO UPDATE SET
			user_id = excluded.user_id,
			opinion_id = excluded.opinion_id,
			section_id = excluded.section_id,
			passage_id = excluded.passage_id,
			start_offset = excluded.start_offset,
			end_offset = excluded.end_offset,
			quote = excluded.quote,
			text = excluded.text,
			asked_at = excluded.asked_at,
			status = excluded.status
	`, question.QuestionID, question.UserID, question.Anchor.OpinionID, question.Anchor.SectionID, question.Anchor.PassageID, question.Anchor.Span.StartOffset, question.Anchor.Span.EndOffset, question.Anchor.Span.Quote, question.Text, question.AskedAt.UTC().Format(time.RFC3339Nano), question.Status.String())
	return err
}

func saveAnswerExec(ctx context.Context, exec sqliteExecutor, answer AnswerDraft) error {
	evidenceJSON, err := json.Marshal(answer.Evidence)
	if err != nil {
		return err
	}
	caveatsJSON, err := json.Marshal(answer.Caveats)
	if err != nil {
		return err
	}
	_, err = exec.ExecContext(ctx, `
		INSERT INTO answers(question_id, answer, evidence_json, caveats_json, generated_at, model_name)
		VALUES(?, ?, ?, ?, ?, ?)
		ON CONFLICT(question_id) DO UPDATE SET
			answer = excluded.answer,
			evidence_json = excluded.evidence_json,
			caveats_json = excluded.caveats_json,
			generated_at = excluded.generated_at,
			model_name = excluded.model_name
	`, answer.QuestionID, answer.Answer, string(evidenceJSON), string(caveatsJSON), answer.GeneratedAt.UTC().Format(time.RFC3339Nano), answer.ModelName)
	return err
}

func loadRawPDFExec(ctx context.Context, exec sqliteExecutor, opinionID OpinionID) (RawPDF, error) {
	var raw RawPDF
	var fetchedAt string
	err := exec.QueryRowContext(ctx, `
		SELECT opinion_id, source_url, bytes, fetched_at, sha256
		FROM raw_pdfs WHERE opinion_id = ?
	`, opinionID).Scan(&raw.OpinionID, &raw.SourceURL, &raw.Bytes, &fetchedAt, &raw.SHA256)
	if errors.Is(err, sql.ErrNoRows) {
		return RawPDF{}, ErrNotFound
	}
	if err != nil {
		return RawPDF{}, err
	}
	raw.FetchedAt, err = time.Parse(time.RFC3339Nano, fetchedAt)
	return raw, err
}

func loadOpinionExec(ctx context.Context, exec sqliteExecutor, opinionID OpinionID) (Opinion, error) {
	var opinion Opinion
	var primaryAuthor sql.NullString
	err := exec.QueryRowContext(ctx, `
		SELECT opinion_id, case_name, docket_number, decided_on, term_label, primary_author, full_text
		FROM opinions WHERE opinion_id = ?
	`, opinionID).Scan(&opinion.OpinionID, &opinion.Meta.CaseName, &opinion.Meta.DocketNumber, &opinion.Meta.DecidedOn, &opinion.Meta.TermLabel, &primaryAuthor, &opinion.FullText)
	if errors.Is(err, sql.ErrNoRows) {
		return Opinion{}, ErrNotFound
	}
	if err != nil {
		return Opinion{}, err
	}
	opinion.Meta.PrimaryAuthor = nullableJustice(primaryAuthor)

	rows, err := exec.QueryContext(ctx, `
		SELECT section_id, kind, title, author, start_page, end_page, text
		FROM sections WHERE opinion_id = ?
		ORDER BY sort_index ASC
	`, opinionID)
	if err != nil {
		return Opinion{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var section Section
		var kind string
		var author sql.NullString
		if err := rows.Scan(&section.SectionID, &kind, &section.Title, &author, &section.StartPage, &section.EndPage, &section.Text); err != nil {
			return Opinion{}, err
		}
		section.OpinionID = opinionID
		section.Kind = SectionKind(kind)
		section.Author = nullableJustice(author)
		opinion.Sections = append(opinion.Sections, section)
	}
	return opinion, rows.Err()
}

func loadPassageExec(ctx context.Context, exec sqliteExecutor, passageID PassageID) (Passage, error) {
	var passage Passage
	var fits int
	err := exec.QueryRowContext(ctx, `
		SELECT passage_id, opinion_id, section_id, sentence_start, sentence_end, page_start, page_end, text, fits_on_screen
		FROM passages WHERE passage_id = ?
	`, passageID).Scan(&passage.PassageID, &passage.OpinionID, &passage.SectionID, &passage.SentenceStart, &passage.SentenceEnd, &passage.PageStart, &passage.PageEnd, &passage.Text, &fits)
	if errors.Is(err, sql.ErrNoRows) {
		return Passage{}, ErrNotFound
	}
	if err != nil {
		return Passage{}, err
	}
	passage.FitsOnScreen = fits == 1

	rows, err := exec.QueryContext(ctx, `
		SELECT citation_id, kind, raw_text, normalized, start_offset, end_offset, quote
		FROM citations WHERE passage_id = ?
		ORDER BY sort_index ASC
	`, passageID)
	if err != nil {
		return Passage{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var citation Citation
		var kind string
		var normalized sql.NullString
		if err := rows.Scan(&citation.CitationID, &kind, &citation.RawText, &normalized, &citation.Span.StartOffset, &citation.Span.EndOffset, &citation.Span.Quote); err != nil {
			return Passage{}, err
		}
		citation.Kind = CitationKind(kind)
		citation.Normalized = nullableString(normalized)
		passage.Citations = append(passage.Citations, citation)
	}
	return passage, rows.Err()
}

func listPassagesExec(ctx context.Context, exec sqliteExecutor, opinionID OpinionID) ([]Passage, error) {
	rows, err := exec.QueryContext(ctx, `
		SELECT passage_id
		FROM passages
		WHERE opinion_id = ?
		ORDER BY page_start ASC, sentence_start ASC, passage_id ASC
	`, opinionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	passageIDs := make([]PassageID, 0)
	for rows.Next() {
		var passageID PassageID
		if err := rows.Scan(&passageID); err != nil {
			return nil, err
		}
		passageIDs = append(passageIDs, passageID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(passageIDs) == 0 {
		return nil, ErrNotFound
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}

	passages := make([]Passage, 0, len(passageIDs))
	for _, passageID := range passageIDs {
		passage, err := loadPassageExec(ctx, exec, passageID)
		if err != nil {
			return nil, err
		}
		passages = append(passages, passage)
	}
	return passages, nil
}

func loadProgressExec(ctx context.Context, exec sqliteExecutor, userID UserID, opinionID OpinionID) (Progress, error) {
	var progress Progress
	var currentPassage sql.NullString
	var updatedAt string
	err := exec.QueryRowContext(ctx, `
		SELECT user_id, opinion_id, current_passage_id, updated_at
		FROM progress WHERE user_id = ? AND opinion_id = ?
	`, userID, opinionID).Scan(&progress.UserID, &progress.OpinionID, &currentPassage, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Progress{}, ErrNotFound
	}
	if err != nil {
		return Progress{}, err
	}
	if currentPassage.Valid {
		id := PassageID(currentPassage.String)
		progress.CurrentPassage = &id
	}
	progress.UpdatedAt, err = time.Parse(time.RFC3339Nano, updatedAt)
	if err != nil {
		return Progress{}, err
	}

	rows, err := exec.QueryContext(ctx, `
		SELECT passage_id FROM completed_passages
		WHERE user_id = ? AND opinion_id = ?
		ORDER BY sort_index ASC
	`, userID, opinionID)
	if err != nil {
		return Progress{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var passageID PassageID
		if err := rows.Scan(&passageID); err != nil {
			return Progress{}, err
		}
		progress.CompletedPassages = append(progress.CompletedPassages, passageID)
	}
	return progress, rows.Err()
}

func loadQuestionsExec(ctx context.Context, exec sqliteExecutor, userID UserID, opinionID OpinionID) ([]Question, error) {
	rows, err := exec.QueryContext(ctx, `
		SELECT question_id, section_id, passage_id, start_offset, end_offset, quote, text, asked_at, status
		FROM questions
		WHERE user_id = ? AND opinion_id = ?
		ORDER BY asked_at ASC
	`, userID, opinionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var questions []Question
	for rows.Next() {
		var question Question
		var askedAt string
		if err := rows.Scan(&question.QuestionID, &question.Anchor.SectionID, &question.Anchor.PassageID, &question.Anchor.Span.StartOffset, &question.Anchor.Span.EndOffset, &question.Anchor.Span.Quote, &question.Text, &askedAt, &question.Status); err != nil {
			return nil, err
		}
		question.UserID = userID
		question.Anchor.OpinionID = opinionID
		question.AskedAt, err = time.Parse(time.RFC3339Nano, askedAt)
		if err != nil {
			return nil, err
		}
		questions = append(questions, question)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(questions) == 0 {
		return nil, ErrNotFound
	}
	return questions, nil
}

func loadAnswersExec(ctx context.Context, exec sqliteExecutor, userID UserID, opinionID OpinionID) ([]AnswerDraft, error) {
	rows, err := exec.QueryContext(ctx, `
		SELECT a.question_id, a.answer, a.evidence_json, a.caveats_json, a.generated_at, a.model_name
		FROM answers a
		JOIN questions q ON q.question_id = a.question_id
		WHERE q.user_id = ? AND q.opinion_id = ?
		ORDER BY q.asked_at ASC
	`, userID, opinionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var answers []AnswerDraft
	for rows.Next() {
		var answer AnswerDraft
		var evidenceJSON string
		var caveatsJSON string
		var generatedAt string
		if err := rows.Scan(&answer.QuestionID, &answer.Answer, &evidenceJSON, &caveatsJSON, &generatedAt, &answer.ModelName); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(evidenceJSON), &answer.Evidence); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(caveatsJSON), &answer.Caveats); err != nil {
			return nil, err
		}
		answer.GeneratedAt, err = time.Parse(time.RFC3339Nano, generatedAt)
		if err != nil {
			return nil, err
		}
		answers = append(answers, answer)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(answers) == 0 {
		return nil, ErrNotFound
	}
	return answers, nil
}

func derefJustice(value *JusticeName) any {
	if value == nil {
		return nil
	}
	return string(*value)
}

func derefString(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func derefPassageID(value *PassageID) any {
	if value == nil {
		return nil
	}
	return string(*value)
}

func nullableJustice(value sql.NullString) *JusticeName {
	if !value.Valid {
		return nil
	}
	justice := JusticeName(value.String)
	return &justice
}

func nullableString(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	copy := value.String
	return &copy
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
