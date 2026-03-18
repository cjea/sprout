package mvp

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestGuessAnswerUsesProviderWhenConfigured(t *testing.T) {
	clock := FixedClock{Time: time.Date(2026, time.March, 17, 23, 0, 0, 0, time.UTC)}
	ctxWindow, question := guessAnswerFixture(t, clock)

	originalClient := http.DefaultClient
	http.DefaultClient = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("got authorization %q", got)
		}
		var request openAIChatCompletionRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if request.Model != "gpt-4.1-mini" {
			t.Fatalf("got model %q", request.Model)
		}
		if len(request.Messages) != 2 {
			t.Fatalf("got %d messages", len(request.Messages))
		}
		userPrompt := request.Messages[1].Content
		for _, fragment := range []string{
			"Full opinion:\nThe statute says one thing. The court explains what compel a contrary finding means in this case.",
			"Active passage:\nThe court explains what compel a contrary finding means in this case.",
			"Selected quote:\ncompel a contrary finding",
			"User question:\nWhat does 'compel a contrary finding' mean in this case?",
		} {
			if !strings.Contains(userPrompt, fragment) {
				t.Fatalf("prompt missing %q", fragment)
			}
		}
		body, err := json.Marshal(openAIChatCompletionResponse{
			Choices: []struct {
				Message openAIChatMessage `json:"message"`
			}{
				{Message: openAIChatMessage{Role: "assistant", Content: "It means the record leaves the agency no room to reject the claimed fact."}},
			},
		})
		if err != nil {
			t.Fatalf("marshal response: %v", err)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(string(body))),
		}, nil
	}),
	}
	defer func() {
		http.DefaultClient = originalClient
	}()

	t.Setenv("OPENAI_API_KEY", "test-key")
	t.Setenv("OPENAI_BASE_URL", "https://provider.test")

	model, err := NewModel("gpt-4.1-mini", 8192)
	if err != nil {
		t.Fatalf("new model: %v", err)
	}
	answer, err := GuessAnswerWithClient(context.Background(), model, ctxWindow, question, clock, nil)
	if err != nil {
		t.Fatalf("guess answer: %v", err)
	}
	if !strings.Contains(string(answer.Answer), "record leaves the agency no room") {
		t.Fatalf("got answer %q", answer.Answer)
	}
	if answer.ModelName != "gpt-4.1-mini" {
		t.Fatalf("got model name %q", answer.ModelName)
	}
}

func TestGuessAnswerFallsBackToHeuristicWithoutProvider(t *testing.T) {
	clock := FixedClock{Time: time.Date(2026, time.March, 17, 23, 5, 0, 0, time.UTC)}
	ctxWindow, question := guessAnswerFixture(t, clock)

	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("OPENAI_BASE_URL", "")

	model, err := NewModel("gpt-4.1-mini", 8192)
	if err != nil {
		t.Fatalf("new model: %v", err)
	}
	answer, err := GuessAnswer(model, ctxWindow, question, clock)
	if err != nil {
		t.Fatalf("guess answer: %v", err)
	}
	if answer.ModelName != "heuristic-v1" {
		t.Fatalf("got model name %q", answer.ModelName)
	}
	if !strings.Contains(string(answer.Answer), "Question:") {
		t.Fatalf("expected heuristic answer text, got %q", answer.Answer)
	}
}

func guessAnswerFixture(t *testing.T, clock FixedClock) (Context, Question) {
	t.Helper()

	userID, _ := NewUserID("reader-1")
	opinionID, _ := NewOpinionID("24-777")
	sectionID, _ := NewSectionID("majority")
	passageID, _ := NewPassageID("p-1")
	meta, _ := NewMeta("Urias-Orellana v. Bondi", "24-777", "2026-03-17", "October Term 2025", nil)
	fullText := Text("The statute says one thing. The court explains what compel a contrary finding means in this case.")
	section, _ := NewSection(opinionID, sectionID, SectionKindMajority, "Majority", nil, 1, 1, fullText)
	opinion, _ := NewOpinion(opinionID, meta, []Section{section}, fullText)
	passage, _ := NewPassage(passageID, opinionID, sectionID, 0, 0, 1, 1, "The court explains what compel a contrary finding means in this case.", nil, true)
	span, _ := NewSpan(24, 49, "compel a contrary finding")
	anchor, _ := NewAnchor(opinionID, sectionID, passageID, span)
	questionID, _ := NewQuestionID("q-1")
	question, _ := NewQuestion(questionID, userID, anchor, "What does 'compel a contrary finding' mean in this case?", clock.Now(), QuestionStatusOpen)
	ctxWindow, _ := GatherContext(opinion, passage, anchor, nil)
	return ctxWindow, question
}
