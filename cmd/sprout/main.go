package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"sprout/mvp"
)

const defaultModelTokens = 8192

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stderr)
		return 2
	}

	switch args[0] {
	case "init-db":
		return runInitDB(args[1:], stdout, stderr)
	case "ingest-url":
		return runIngestURL(args[1:], stdout, stderr)
	case "ingest-file":
		return runIngestFile(args[1:], stdout, stderr)
	case "show-opinion":
		return runShowOpinion(args[1:], stdout, stderr)
	case "list-passages":
		return runListPassages(args[1:], stdout, stderr)
	case "show-passage":
		return runShowPassage(args[1:], stdout, stderr)
	case "show-progress":
		return runShowProgress(args[1:], stdout, stderr)
	case "read":
		return runRead(args[1:], stdout, stderr)
	case "complete-passage":
		return runCompletePassage(args[1:], stdout, stderr)
	case "ask":
		return runAsk(args[1:], stdout, stderr)
	case "help", "-h", "--help":
		printUsage(stdout)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n\n", args[0])
		printUsage(stderr)
		return 2
	}
}

func runInitDB(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("init-db", flag.ContinueOnError)
	flags.SetOutput(stderr)
	dbPath := flags.String("db", "var/sprout.db", "sqlite database path")
	if err := flags.Parse(args); err != nil {
		return 2
	}

	storage, err := openStorage(*dbPath)
	if err != nil {
		return fail(stderr, err)
	}
	defer storage.Close()

	fmt.Fprintf(stdout, "db=%s\nstatus=ready\n", *dbPath)
	return 0
}

func runIngestURL(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("ingest-url", flag.ContinueOnError)
	flags.SetOutput(stderr)
	dbPath := flags.String("db", "var/sprout.db", "sqlite database path")
	userValue := flags.String("user", "demo", "user id")
	urlValue := flags.String("url", "", "opinion pdf url")
	modelValue := flags.String("model", "heuristic-v1", "model name for guess functions")
	if err := flags.Parse(args); err != nil {
		return 2
	}

	userID, input, model, err := parseRuntimeInputs(*userValue, *urlValue, *modelValue)
	if err != nil {
		return fail(stderr, err)
	}
	storage, err := openStorage(*dbPath)
	if err != nil {
		return fail(stderr, err)
	}
	defer storage.Close()

	state, err := mvp.RunMVP(context.Background(), userID, input, http.DefaultClient, storage, mvp.SystemClock{}, model)
	if err != nil {
		return fail(stderr, err)
	}
	printIngestResult(stdout, state)
	return 0
}

func runIngestFile(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("ingest-file", flag.ContinueOnError)
	flags.SetOutput(stderr)
	dbPath := flags.String("db", "var/sprout.db", "sqlite database path")
	userValue := flags.String("user", "demo", "user id")
	filePath := flags.String("file", "", "local pdf path")
	urlValue := flags.String("url", "", "source url for the pdf")
	modelValue := flags.String("model", "heuristic-v1", "model name for guess functions")
	if err := flags.Parse(args); err != nil {
		return 2
	}

	if strings.TrimSpace(*filePath) == "" {
		return fail(stderr, errors.New("file is required"))
	}

	userID, sourceURL, model, err := parseIngestFileInputs(*userValue, *urlValue, *modelValue)
	if err != nil {
		return fail(stderr, err)
	}

	storage, err := openStorage(*dbPath)
	if err != nil {
		return fail(stderr, err)
	}
	defer storage.Close()

	bytes, err := os.ReadFile(*filePath)
	if err != nil {
		return fail(stderr, fmt.Errorf("read file: %w", err))
	}
	opinionID, err := mvp.MakeOpinionID(sourceURL)
	if err != nil {
		return fail(stderr, err)
	}
	raw, err := mvp.MakeRawPDF(opinionID, sourceURL, bytes, mvp.SystemClock{}.Now())
	if err != nil {
		return fail(stderr, err)
	}
	firstPassageID, err := mvp.IngestRawPDF(userID, raw, storage, model)
	if err != nil {
		return fail(stderr, err)
	}
	state, err := mvp.OpenPassage(userID, firstPassageID, storage)
	if err != nil {
		return fail(stderr, err)
	}
	printIngestResult(stdout, state)
	return 0
}

func runShowOpinion(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("show-opinion", flag.ContinueOnError)
	flags.SetOutput(stderr)
	dbPath := flags.String("db", "var/sprout.db", "sqlite database path")
	opinionValue := flags.String("opinion-id", "", "opinion id")
	if err := flags.Parse(args); err != nil {
		return 2
	}

	opinionID, err := mvp.NewOpinionID(*opinionValue)
	if err != nil {
		return fail(stderr, err)
	}
	storage, err := openStorage(*dbPath)
	if err != nil {
		return fail(stderr, err)
	}
	defer storage.Close()

	opinion, err := loadOpinion(storage, opinionID)
	if err != nil {
		return fail(stderr, err)
	}

	fmt.Fprintf(stdout, "opinion_id=%s\ncase_name=%s\ndocket_number=%s\ndecided_on=%s\nterm=%s\nsections=%d\n",
		opinion.OpinionID,
		opinion.Meta.CaseName,
		opinion.Meta.DocketNumber,
		opinion.Meta.DecidedOn,
		opinion.Meta.TermLabel,
		len(opinion.Sections),
	)
	for _, section := range opinion.Sections {
		fmt.Fprintf(stdout, "section=%s kind=%s pages=%d-%d title=%q\n",
			section.SectionID,
			section.Kind,
			section.StartPage,
			section.EndPage,
			section.Title,
		)
	}
	return 0
}

func runListPassages(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("list-passages", flag.ContinueOnError)
	flags.SetOutput(stderr)
	dbPath := flags.String("db", "var/sprout.db", "sqlite database path")
	opinionValue := flags.String("opinion-id", "", "opinion id")
	if err := flags.Parse(args); err != nil {
		return 2
	}

	opinionID, err := mvp.NewOpinionID(*opinionValue)
	if err != nil {
		return fail(stderr, err)
	}
	storage, err := openStorage(*dbPath)
	if err != nil {
		return fail(stderr, err)
	}
	defer storage.Close()

	passages, err := listPassages(storage, opinionID)
	if err != nil {
		return fail(stderr, err)
	}

	for _, passage := range passages {
		fmt.Fprintf(stdout, "passage_id=%s section_id=%s pages=%d-%d sentences=%d-%d fits=%t text=%q\n",
			passage.PassageID,
			passage.SectionID,
			passage.PageStart,
			passage.PageEnd,
			passage.SentenceStart,
			passage.SentenceEnd,
			passage.FitsOnScreen,
			passage.Text,
		)
	}
	return 0
}

func runShowPassage(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("show-passage", flag.ContinueOnError)
	flags.SetOutput(stderr)
	dbPath := flags.String("db", "var/sprout.db", "sqlite database path")
	passageValue := flags.String("passage-id", "", "passage id")
	if err := flags.Parse(args); err != nil {
		return 2
	}

	passageID, err := mvp.NewPassageID(*passageValue)
	if err != nil {
		return fail(stderr, err)
	}
	storage, err := openStorage(*dbPath)
	if err != nil {
		return fail(stderr, err)
	}
	defer storage.Close()

	passage, err := loadPassage(storage, passageID)
	if err != nil {
		return fail(stderr, err)
	}
	printPassage(stdout, passage)
	return 0
}

func runShowProgress(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("show-progress", flag.ContinueOnError)
	flags.SetOutput(stderr)
	dbPath := flags.String("db", "var/sprout.db", "sqlite database path")
	userValue := flags.String("user", "demo", "user id")
	opinionValue := flags.String("opinion-id", "", "opinion id")
	if err := flags.Parse(args); err != nil {
		return 2
	}

	userID, err := mvp.NewUserID(*userValue)
	if err != nil {
		return fail(stderr, err)
	}
	opinionID, err := mvp.NewOpinionID(*opinionValue)
	if err != nil {
		return fail(stderr, err)
	}
	storage, err := openStorage(*dbPath)
	if err != nil {
		return fail(stderr, err)
	}
	defer storage.Close()

	progress, err := loadProgress(storage, userID, opinionID)
	if err != nil {
		return fail(stderr, err)
	}
	currentPassage := ""
	if progress.CurrentPassage != nil {
		currentPassage = string(*progress.CurrentPassage)
	}
	fmt.Fprintf(stdout, "user_id=%s\nopinion_id=%s\ncurrent_passage_id=%s\ncompleted=%d\nopen_questions=%d\nupdated_at=%s\n",
		progress.UserID,
		progress.OpinionID,
		currentPassage,
		len(progress.CompletedPassages),
		len(progress.OpenQuestionIDs),
		progress.UpdatedAt.Format(timeLayout()),
	)
	return 0
}

func runRead(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("read", flag.ContinueOnError)
	flags.SetOutput(stderr)
	dbPath := flags.String("db", "var/sprout.db", "sqlite database path")
	userValue := flags.String("user", "demo", "user id")
	opinionValue := flags.String("opinion-id", "", "opinion id")
	if err := flags.Parse(args); err != nil {
		return 2
	}

	userID, err := mvp.NewUserID(*userValue)
	if err != nil {
		return fail(stderr, err)
	}
	opinionID, err := mvp.NewOpinionID(*opinionValue)
	if err != nil {
		return fail(stderr, err)
	}
	storage, err := openStorage(*dbPath)
	if err != nil {
		return fail(stderr, err)
	}
	defer storage.Close()

	state, err := mvp.ResumeReading(userID, opinionID, storage)
	if err != nil {
		return fail(stderr, err)
	}

	fmt.Fprintf(stdout, "case_name=%s\n", state.Opinion.Meta.CaseName)
	printPassage(stdout, state.Passage)
	fmt.Fprintf(stdout, "completed=%d\nopen_questions=%d\n", len(state.Progress.CompletedPassages), len(state.Progress.OpenQuestionIDs))
	return 0
}

func runCompletePassage(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("complete-passage", flag.ContinueOnError)
	flags.SetOutput(stderr)
	dbPath := flags.String("db", "var/sprout.db", "sqlite database path")
	userValue := flags.String("user", "demo", "user id")
	passageValue := flags.String("passage-id", "", "passage id")
	if err := flags.Parse(args); err != nil {
		return 2
	}

	userID, err := mvp.NewUserID(*userValue)
	if err != nil {
		return fail(stderr, err)
	}
	passageID, err := mvp.NewPassageID(*passageValue)
	if err != nil {
		return fail(stderr, err)
	}
	storage, err := openStorage(*dbPath)
	if err != nil {
		return fail(stderr, err)
	}
	defer storage.Close()

	progress, err := mvp.CompletePassage(userID, passageID, storage, mvp.SystemClock{})
	if err != nil {
		return fail(stderr, err)
	}
	fmt.Fprintf(stdout, "passage_id=%s\ncompleted=%d\nupdated_at=%s\n", passageID, len(progress.CompletedPassages), progress.UpdatedAt.Format(timeLayout()))
	return 0
}

func runAsk(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("ask", flag.ContinueOnError)
	flags.SetOutput(stderr)
	dbPath := flags.String("db", "var/sprout.db", "sqlite database path")
	userValue := flags.String("user", "demo", "user id")
	passageValue := flags.String("passage-id", "", "passage id")
	startValue := flags.Int("start", 0, "selected span start offset")
	endValue := flags.Int("end", 0, "selected span end offset")
	questionValue := flags.String("question", "", "question text")
	modelValue := flags.String("model", "heuristic-v1", "model name for guess functions")
	if err := flags.Parse(args); err != nil {
		return 2
	}

	userID, err := mvp.NewUserID(*userValue)
	if err != nil {
		return fail(stderr, err)
	}
	passageID, err := mvp.NewPassageID(*passageValue)
	if err != nil {
		return fail(stderr, err)
	}
	model, err := mvp.NewModel(*modelValue, defaultModelTokens)
	if err != nil {
		return fail(stderr, err)
	}
	storage, err := openStorage(*dbPath)
	if err != nil {
		return fail(stderr, err)
	}
	defer storage.Close()

	passage, err := loadPassage(storage, passageID)
	if err != nil {
		return fail(stderr, err)
	}
	opinion, err := loadOpinion(storage, passage.OpinionID)
	if err != nil {
		return fail(stderr, err)
	}
	span, err := mvp.SelectSpan(passage, mvp.Offset(*startValue), mvp.Offset(*endValue))
	if err != nil {
		return fail(stderr, err)
	}
	anchor, err := mvp.AnchorSpan(passage.OpinionID, passage.SectionID, passage.PassageID, span)
	if err != nil {
		return fail(stderr, err)
	}
	question, err := mvp.AskQuestion(userID, anchor, mvp.QuestionText(*questionValue), mvp.SystemClock{})
	if err != nil {
		return fail(stderr, err)
	}
	question, err = mvp.SaveQuestion(storage, question)
	if err != nil {
		return fail(stderr, err)
	}
	questions, err := loadQuestions(storage, userID, passage.OpinionID)
	if err != nil && !errors.Is(err, mvp.ErrNotFound) {
		return fail(stderr, err)
	}
	contextValue, err := mvp.GatherContext(opinion, passage, anchor, questions)
	if err != nil {
		return fail(stderr, err)
	}
	answer, err := mvp.GuessAnswer(model, contextValue, question, mvp.SystemClock{})
	if err != nil {
		return fail(stderr, err)
	}

	fmt.Fprintf(stdout, "question_id=%s\nanswer=%q\n", answer.QuestionID, answer.Answer)
	for _, evidence := range answer.Evidence {
		fmt.Fprintf(stdout, "evidence label=%q quote=%q passage_id=%s\n", evidence.Label, evidence.Quote, evidence.Anchor.PassageID)
	}
	for _, caveat := range answer.Caveats {
		fmt.Fprintf(stdout, "caveat=%q\n", caveat)
	}
	return 0
}

func parseRuntimeInputs(userValue, urlValue, modelValue string) (mvp.UserID, mvp.UserInput, mvp.Model, error) {
	userID, err := mvp.NewUserID(userValue)
	if err != nil {
		return "", mvp.UserInput{}, mvp.Model{}, err
	}
	sourceURL, err := mvp.EnterURL(urlValue)
	if err != nil {
		return "", mvp.UserInput{}, mvp.Model{}, err
	}
	input, err := mvp.NewUserInput(sourceURL)
	if err != nil {
		return "", mvp.UserInput{}, mvp.Model{}, err
	}
	model, err := mvp.NewModel(modelValue, defaultModelTokens)
	if err != nil {
		return "", mvp.UserInput{}, mvp.Model{}, err
	}
	return userID, input, model, nil
}

func parseIngestFileInputs(userValue, urlValue, modelValue string) (mvp.UserID, mvp.URL, mvp.Model, error) {
	userID, err := mvp.NewUserID(userValue)
	if err != nil {
		return "", "", mvp.Model{}, err
	}
	sourceURL, err := mvp.EnterURL(urlValue)
	if err != nil {
		return "", "", mvp.Model{}, err
	}
	model, err := mvp.NewModel(modelValue, defaultModelTokens)
	if err != nil {
		return "", "", mvp.Model{}, err
	}
	return userID, sourceURL, model, nil
}

func openStorage(dbPath string) (*mvp.SQLiteStorage, error) {
	dir := filepath.Dir(dbPath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create db directory: %w", err)
		}
	}
	return mvp.OpenSQLiteWithMigrations(dbPath, mvp.SystemClock{})
}

func printIngestResult(stdout io.Writer, state mvp.ReadingState) {
	fmt.Fprintf(stdout, "opinion_id=%s\ncase_name=%s\nsection_count=%d\ncurrent_passage_id=%s\n",
		state.Opinion.OpinionID,
		state.Opinion.Meta.CaseName,
		len(state.Opinion.Sections),
		state.Passage.PassageID,
	)
	printPassage(stdout, state.Passage)
}

func printPassage(stdout io.Writer, passage mvp.Passage) {
	fmt.Fprintf(stdout, "passage_id=%s\nsection_id=%s\npages=%d-%d\ntext=%q\n",
		passage.PassageID,
		passage.SectionID,
		passage.PageStart,
		passage.PageEnd,
		passage.Text,
	)
	for _, citation := range passage.Citations {
		fmt.Fprintf(stdout, "citation kind=%s raw=%q\n", citation.Kind, citation.RawText)
	}
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "sprout <command> [flags]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  init-db            Create the SQLite database and apply migrations")
	fmt.Fprintln(w, "  ingest-url         Fetch a Supreme Court opinion PDF from a URL and ingest it")
	fmt.Fprintln(w, "  ingest-file        Ingest a local PDF using a source URL for provenance")
	fmt.Fprintln(w, "  show-opinion       Print stored opinion metadata and section summary")
	fmt.Fprintln(w, "  list-passages      Print all stored passages for an opinion")
	fmt.Fprintln(w, "  show-passage       Print one stored passage and its citations")
	fmt.Fprintln(w, "  show-progress      Print reading progress for a user and opinion")
	fmt.Fprintln(w, "  read               Resume the current passage for a user and opinion")
	fmt.Fprintln(w, "  complete-passage   Mark a passage complete for a user")
	fmt.Fprintln(w, "  ask                Save a question against a span and print a guessed answer")
}

func fail(stderr io.Writer, err error) int {
	fmt.Fprintf(stderr, "error: %v\n", err)
	return 1
}

func loadOpinion(storage *mvp.SQLiteStorage, opinionID mvp.OpinionID) (mvp.Opinion, error) {
	record, err := storage.LoadOpinion(opinionID)
	if err != nil {
		return mvp.Opinion{}, err
	}
	opinion, ok := record.(mvp.Opinion)
	if !ok {
		return mvp.Opinion{}, mvp.ErrWrongRecordType
	}
	return opinion, nil
}

func loadPassage(storage *mvp.SQLiteStorage, passageID mvp.PassageID) (mvp.Passage, error) {
	record, err := storage.LoadPassage(passageID)
	if err != nil {
		return mvp.Passage{}, err
	}
	passage, ok := record.(mvp.Passage)
	if !ok {
		return mvp.Passage{}, mvp.ErrWrongRecordType
	}
	return passage, nil
}

func loadProgress(storage *mvp.SQLiteStorage, userID mvp.UserID, opinionID mvp.OpinionID) (mvp.Progress, error) {
	record, err := storage.LoadProgress(userID, opinionID)
	if err != nil {
		return mvp.Progress{}, err
	}
	progress, ok := record.(mvp.Progress)
	if !ok {
		return mvp.Progress{}, mvp.ErrWrongRecordType
	}
	return progress, nil
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

func timeLayout() string {
	return "2006-01-02T15:04:05.999999999Z07:00"
}
