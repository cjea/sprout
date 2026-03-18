package mvp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

var (
	ErrGuessAnswerChoiceMissing = errors.New("guess answer response contained no choices")
)

const defaultGuessAnswerBaseURL = "https://api.openai.com"

type OpenAIAnswerGuesser struct {
	Model      Model
	Clock      Clock
	HTTPClient *http.Client
	BaseURL    string
	APIKey     string
}

type openAIChatCompletionRequest struct {
	Model       string              `json:"model"`
	Messages    []openAIChatMessage `json:"messages"`
	Temperature float64             `json:"temperature"`
}

type openAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIChatCompletionResponse struct {
	Choices []struct {
		Message openAIChatMessage `json:"message"`
	} `json:"choices"`
}

func GuessAnswerWithClient(
	ctx context.Context,
	model Model,
	contextValue Context,
	question Question,
	clock Clock,
	guesser AnswerGuesser,
) (AnswerDraft, error) {
	if guesser == nil {
		return GuessAnswer(model, contextValue, question, clock)
	}
	return guesser.GuessAnswer(ctx, contextValue, question)
}

func GuessAnswer(model Model, contextValue Context, question Question, clock Clock) (AnswerDraft, error) {
	if usesHeuristicAnswerGuesser(model) {
		guesser := HeuristicAnswerGuesser{Model: model, Clock: clock}
		return guesser.GuessAnswer(context.Background(), contextValue, question)
	}

	guesser, ok := answerGuesserFromEnv(model, clock)
	if !ok {
		heuristic := HeuristicAnswerGuesser{Model: Model{Name: "heuristic-v1", MaxContextTokens: model.MaxContextTokens}, Clock: clock}
		return heuristic.GuessAnswer(context.Background(), contextValue, question)
	}

	draft, err := guesser.GuessAnswer(context.Background(), contextValue, question)
	if err == nil {
		return draft, nil
	}

	heuristic := HeuristicAnswerGuesser{Model: Model{Name: "heuristic-v1", MaxContextTokens: model.MaxContextTokens}, Clock: clock}
	fallback, fallbackErr := heuristic.GuessAnswer(context.Background(), contextValue, question)
	if fallbackErr != nil {
		return AnswerDraft{}, err
	}
	fallback.Caveats = append([]string{
		fmt.Sprintf("Model guess failed and fell back to heuristic answer: %v", err),
	}, fallback.Caveats...)
	return fallback, nil
}

func usesHeuristicAnswerGuesser(model Model) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(model.Name)), "heuristic")
}

func answerGuesserFromEnv(model Model, clock Clock) (AnswerGuesser, bool) {
	apiKey := strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	if apiKey == "" {
		return nil, false
	}
	baseURL := strings.TrimSpace(os.Getenv("OPENAI_BASE_URL"))
	if baseURL == "" {
		baseURL = defaultGuessAnswerBaseURL
	}
	return OpenAIAnswerGuesser{
		Model:      model,
		Clock:      clock,
		HTTPClient: http.DefaultClient,
		BaseURL:    strings.TrimRight(baseURL, "/"),
		APIKey:     apiKey,
	}, true
}

func (g OpenAIAnswerGuesser) GuessAnswer(ctx context.Context, contextValue Context, question Question) (AnswerDraft, error) {
	if err := g.Model.Validate(); err != nil {
		return AnswerDraft{}, err
	}
	if err := contextValue.Validate(); err != nil {
		return AnswerDraft{}, err
	}
	if err := question.Validate(); err != nil {
		return AnswerDraft{}, err
	}
	if strings.TrimSpace(g.APIKey) == "" {
		return AnswerDraft{}, ErrEmptyModelName
	}

	requestBody := openAIChatCompletionRequest{
		Model:       g.Model.Name,
		Messages:    buildGuessAnswerMessages(contextValue, question),
		Temperature: 0.2,
	}
	payload, err := json.Marshal(requestBody)
	if err != nil {
		return AnswerDraft{}, err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, g.BaseURL+"/v1/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return AnswerDraft{}, err
	}
	request.Header.Set("Authorization", "Bearer "+g.APIKey)
	request.Header.Set("Content-Type", "application/json")

	client := g.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	response, err := client.Do(request)
	if err != nil {
		return AnswerDraft{}, err
	}
	defer response.Body.Close()
	if response.StatusCode >= 300 {
		return AnswerDraft{}, fmt.Errorf("guess answer provider returned status %d", response.StatusCode)
	}

	var completion openAIChatCompletionResponse
	if err := json.NewDecoder(response.Body).Decode(&completion); err != nil {
		return AnswerDraft{}, err
	}
	if len(completion.Choices) == 0 {
		return AnswerDraft{}, ErrGuessAnswerChoiceMissing
	}
	content := strings.TrimSpace(completion.Choices[0].Message.Content)
	evidence, err := NewEvidence(question.Anchor, question.Anchor.Span.Quote, "selected passage")
	if err != nil {
		return AnswerDraft{}, err
	}
	return NewAnswerDraft(
		question.QuestionID,
		AnswerText(content),
		[]Evidence{evidence},
		[]string{"This is a model guess grounded in the full opinion and the active passage."},
		guessClockOrNow(g.Clock),
		g.Model.Name,
	)
}

func buildGuessAnswerMessages(contextValue Context, question Question) []openAIChatMessage {
	return []openAIChatMessage{
		{
			Role: "system",
			Content: strings.TrimSpace(`You answer questions about U.S. Supreme Court opinions.
Use the source opinion and the active passage.
Answer directly and conservatively.
If the question asks about a phrase, explain what it means in this case.
Do not claim certainty beyond the source text.`),
		},
		{
			Role: "user",
			Content: strings.TrimSpace(fmt.Sprintf(
				"Case: %s\nDocket: %s\nSection: %s\n\nFull opinion:\n%s\n\nActive passage:\n%s\n\nSelected quote:\n%s\n\nUser question:\n%s",
				contextValue.Opinion.Meta.CaseName,
				contextValue.Opinion.Meta.DocketNumber,
				contextValue.ActivePassage.SectionID,
				contextValue.Opinion.FullText,
				contextValue.ActivePassage.Text,
				question.Anchor.Span.Quote,
				question.Text,
			)),
		},
	}
}

func guessClockOrNow(clock Clock) time.Time {
	if clock == nil {
		return time.Now().UTC()
	}
	return clock.Now()
}
