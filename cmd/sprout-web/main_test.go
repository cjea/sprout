package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestServerServesShell(t *testing.T) {
	handler, closeFn, err := newServer(testServerConfig(t))
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	defer closeFn()

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d", recorder.Code, http.StatusOK)
	}
	body := recorder.Body.String()
	for _, fragment := range []string{
		"Supreme Court Opinion Reader",
		"workspace-shell",
		"passage-stage",
		"secondary-surface",
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("response missing %q", fragment)
		}
	}
}

func TestServerServesAssets(t *testing.T) {
	handler, closeFn, err := newServer(testServerConfig(t))
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	defer closeFn()

	tests := []struct {
		path     string
		contains []string
	}{
		{path: "/app.css", contains: []string{"--paper", "@media (max-width: 920px)", ".passage-stage"}},
		{path: "/app.js", contains: []string{"setPanel", "popstate", "/api/question"}},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, tt.path, nil)
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)

			if recorder.Code != http.StatusOK {
				t.Fatalf("got status %d, want %d", recorder.Code, http.StatusOK)
			}
			for _, fragment := range tt.contains {
				if !strings.Contains(recorder.Body.String(), fragment) {
					t.Fatalf("response missing %q", fragment)
				}
			}
		})
	}
}

func TestServerRejectsUnknownPaths(t *testing.T) {
	handler, closeFn, err := newServer(testServerConfig(t))
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	defer closeFn()

	request := httptest.NewRequest(http.MethodGet, "/missing", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("got status %d, want %d", recorder.Code, http.StatusNotFound)
	}
}

func TestReaderAPIUsesRealFixtureData(t *testing.T) {
	handler, closeFn, err := newServer(testServerConfig(t))
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	defer closeFn()

	request := httptest.NewRequest(http.MethodGet, "/api/reader", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response readerResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.Opinion.OpinionID != "24-777_9ol1" {
		t.Fatalf("got opinion id %q", response.Opinion.OpinionID)
	}
	if response.Opinion.CaseName != "Urias-Orellana v. Bondi" {
		t.Fatalf("got case name %q", response.Opinion.CaseName)
	}
	if response.Passage.PassageID == "" {
		t.Fatal("expected current passage")
	}
	if len(response.Passages) == 0 {
		t.Fatal("expected passage list")
	}
}

func TestReaderAPICanOpenSpecificPassage(t *testing.T) {
	handler, closeFn, err := newServer(testServerConfig(t))
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	defer closeFn()

	firstRequest := httptest.NewRequest(http.MethodGet, "/api/reader", nil)
	firstRecorder := httptest.NewRecorder()
	handler.ServeHTTP(firstRecorder, firstRequest)

	if firstRecorder.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d", firstRecorder.Code, http.StatusOK)
	}

	var first readerResponse
	if err := json.Unmarshal(firstRecorder.Body.Bytes(), &first); err != nil {
		t.Fatalf("decode first response: %v", err)
	}
	if len(first.Passages) < 2 {
		t.Fatalf("need at least two passages, got %d", len(first.Passages))
	}

	secondPassageID := first.Passages[1].PassageID
	secondRequest := httptest.NewRequest(http.MethodGet, "/api/reader?passage="+secondPassageID, nil)
	secondRecorder := httptest.NewRecorder()
	handler.ServeHTTP(secondRecorder, secondRequest)

	if secondRecorder.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d body=%s", secondRecorder.Code, http.StatusOK, secondRecorder.Body.String())
	}

	var second readerResponse
	if err := json.Unmarshal(secondRecorder.Body.Bytes(), &second); err != nil {
		t.Fatalf("decode second response: %v", err)
	}
	if second.Passage.PassageID != secondPassageID {
		t.Fatalf("got passage %q, want %q", second.Passage.PassageID, secondPassageID)
	}
	if second.Progress.CurrentPassageID != secondPassageID {
		t.Fatalf("got current passage %q, want %q", second.Progress.CurrentPassageID, secondPassageID)
	}
}

func TestReaderAPINormalizesInvalidPassageToSafeState(t *testing.T) {
	handler, closeFn, err := newServer(testServerConfig(t))
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	defer closeFn()

	request := httptest.NewRequest(http.MethodGet, "/api/reader?passage=missing-passage", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response readerResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Passage.PassageID == "" {
		t.Fatal("expected normalized passage")
	}
}

func TestCompleteEndpointUpdatesProgress(t *testing.T) {
	handler, closeFn, err := newServer(testServerConfig(t))
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	defer closeFn()

	reader := mustReadReader(t, handler, "/api/reader")
	payload := completeRequest{
		UserID:    defaultUserID,
		OpinionID: reader.Opinion.OpinionID,
		PassageID: reader.Passage.PassageID,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/complete", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var progress progressDTO
	if err := json.Unmarshal(recorder.Body.Bytes(), &progress); err != nil {
		t.Fatalf("decode progress: %v", err)
	}
	if len(progress.CompletedPassages) == 0 {
		t.Fatal("expected completed passage")
	}
	if progress.CompletedPassages[0] != reader.Passage.PassageID {
		t.Fatalf("got completed passage %q, want %q", progress.CompletedPassages[0], reader.Passage.PassageID)
	}
}

func TestQuestionEndpointReturnsQuestionAndAnswer(t *testing.T) {
	handler, closeFn, err := newServer(testServerConfig(t))
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	defer closeFn()

	reader := mustReadReader(t, handler, "/api/reader")
	end := 32
	if len(reader.Passage.Text) < end {
		end = len(reader.Passage.Text)
	}
	payload := questionRequest{
		UserID:    defaultUserID,
		OpinionID: reader.Opinion.OpinionID,
		PassageID: reader.Passage.PassageID,
		Start:     0,
		End:       end,
		Text:      "What does this sentence mean?",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/question", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response questionResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Question.QuestionID == "" {
		t.Fatal("expected question id")
	}
	if response.Question.Quote == "" {
		t.Fatal("expected anchored quote")
	}
	if response.Answer.Answer == "" {
		t.Fatal("expected answer text")
	}
	if len(response.Answer.Caveats) == 0 {
		t.Fatal("expected caveat")
	}
}

func testServerConfig(t *testing.T) serverConfig {
	t.Helper()

	return serverConfig{
		DBPath:      filepath.Join(t.TempDir(), "browser.db"),
		FixturePath: filepath.Join("..", "..", "fixtures", "scotus", "24-777_9ol1.pdf"),
		FixtureURL:  defaultFixtureURL,
		UserID:      defaultUserID,
		ModelName:   defaultModelName,
	}
}

func mustReadReader(t *testing.T, handler http.Handler, path string) readerResponse {
	t.Helper()

	request := httptest.NewRequest(http.MethodGet, path, nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	var response readerResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return response
}
