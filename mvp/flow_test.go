package mvp

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestReadingAndQuestionFlow(t *testing.T) {
	storage := NewMemoryStorage()
	clock := FixedClock{Time: time.Date(2026, time.March, 16, 23, 30, 0, 0, time.UTC)}
	userID, _ := NewUserID("reader-1")
	opinionID, _ := NewOpinionID("24-777")
	sectionID, _ := NewSectionID("majority")
	passageID, _ := NewPassageID("p-1")

	meta, _ := NewMeta("Case", "24-777", "2026-03-16", "2025", nil)
	section, _ := NewSection(opinionID, sectionID, SectionKindMajority, "Majority", nil, 1, 1, "One sentence about Roe v. Wade.")
	opinion, _ := NewOpinion(opinionID, meta, []Section{section}, "One sentence about Roe v. Wade.")
	passage, _ := NewPassage(passageID, opinionID, sectionID, 0, 0, 1, 1, "One sentence about Roe v. Wade.", nil, true)
	passageWithCitations, _ := AttachCitations([]Passage{passage})

	if _, err := saveOpinion(storage, opinion); err != nil {
		t.Fatalf("save opinion: %v", err)
	}
	if _, err := savePassages(storage, passageWithCitations); err != nil {
		t.Fatalf("save passage: %v", err)
	}

	queue, err := BuildQueue(userID, opinion, passageWithCitations)
	if err != nil {
		t.Fatalf("build queue: %v", err)
	}
	next, err := NextPassage(queue)
	if err != nil {
		t.Fatalf("next passage: %v", err)
	}

	state, err := OpenPassage(userID, *next, storage)
	if err != nil {
		t.Fatalf("open passage: %v", err)
	}
	span, err := SelectSpan(state.Passage, 19, 31)
	if err != nil {
		t.Fatalf("select span: %v", err)
	}
	anchor, err := AnchorSpan(state.Opinion.OpinionID, state.Passage.SectionID, state.Passage.PassageID, span)
	if err != nil {
		t.Fatalf("anchor span: %v", err)
	}
	question, err := AskQuestion(userID, anchor, "What does this precedent do?", clock)
	if err != nil {
		t.Fatalf("ask question: %v", err)
	}
	if _, err := SaveQuestion(storage, question); err != nil {
		t.Fatalf("save question: %v", err)
	}

	questions, err := loadQuestions(storage, userID, opinionID)
	if err != nil {
		t.Fatalf("load questions: %v", err)
	}
	context, err := GatherContext(opinion, state.Passage, anchor, questions)
	if err != nil {
		t.Fatalf("gather context: %v", err)
	}
	model, _ := NewModel("gpt-5", 8000)
	draft, err := GuessAnswer(model, context, question, clock)
	if err != nil {
		t.Fatalf("guess answer: %v", err)
	}
	if draft.QuestionID != question.QuestionID {
		t.Fatalf("got answer for question %q, want %q", draft.QuestionID, question.QuestionID)
	}

	progress, err := CompletePassage(userID, state.Passage.PassageID, storage, clock)
	if err != nil {
		t.Fatalf("complete passage: %v", err)
	}
	if len(progress.CompletedPassages) != 1 {
		t.Fatalf("expected one completed passage")
	}

	if _, err := ResumeReading(userID, opinionID, storage); err != nil {
		t.Fatalf("resume reading: %v", err)
	}
}

func TestRunMVP(t *testing.T) {
	fixture := loadRealFixturePDF(t)
	client := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(fixture.Bytes)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	userID, _ := NewUserID("reader-1")
	input, _ := NewUserInput(URL("https://www.supremecourt.gov/opinions/25pdf/24-777_9ol1.pdf"))
	model, _ := NewModel("gpt-5", 8000)
	storage := NewMemoryStorage()
	clock := FixedClock{Time: time.Date(2026, time.March, 16, 23, 45, 0, 0, time.UTC)}

	state, err := RunMVP(context.Background(), userID, input, client, storage, clock, model)
	if err != nil {
		t.Fatalf("run mvp: %v", err)
	}
	if state.Passage.PassageID == "" {
		t.Fatalf("expected reading state to include a passage")
	}
}

func TestHeuristicAnswerGuesserImplementsInterface(t *testing.T) {
	clock := FixedClock{Time: time.Date(2026, time.March, 17, 1, 10, 0, 0, time.UTC)}
	userID, _ := NewUserID("reader-1")
	opinionID, _ := NewOpinionID("24-777")
	sectionID, _ := NewSectionID("majority")
	passageID, _ := NewPassageID("p-1")

	meta, _ := NewMeta("Case", "24-777", "2026-03-16", "2025", nil)
	section, _ := NewSection(opinionID, sectionID, SectionKindMajority, "Majority", nil, 1, 1, "One sentence about Roe v. Wade.")
	opinion, _ := NewOpinion(opinionID, meta, []Section{section}, "One sentence about Roe v. Wade.")
	passage, _ := NewPassage(passageID, opinionID, sectionID, 0, 0, 1, 1, "One sentence about Roe v. Wade.", nil, true)
	passageWithCitations, _ := AttachCitations([]Passage{passage})
	span, _ := NewSpan(19, 31, "Roe v. Wade")
	anchor, _ := NewAnchor(opinionID, sectionID, passageID, span)
	question, _ := NewQuestion(QuestionID("q-1"), userID, anchor, "What does this precedent do?", clock.Now(), QuestionStatusOpen)
	ctxWindow, _ := GatherContext(opinion, passageWithCitations[0], anchor, nil)
	model, _ := NewModel("gpt-5", 8000)

	var guesser AnswerGuesser = HeuristicAnswerGuesser{Model: model, Clock: clock}
	draft, err := guesser.GuessAnswer(context.Background(), ctxWindow, question)
	if err != nil {
		t.Fatalf("guess answer via interface: %v", err)
	}
	if draft.ModelName != "gpt-5" {
		t.Fatalf("got model name %q", draft.ModelName)
	}
}
