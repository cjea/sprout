package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"sprout/mvp"
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
		"Passage Repair",
		"mergeNext",
		"data-previous-passage",
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
		{path: "/app.js", contains: []string{"setPanel", "popstate", "/api/question", "/api/repair/apply", "mergeNext", "data-previous-passage", "previousPassageId", "openPassage"}},
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

func TestReaderAPIDirectOpenRetainsCanonicalPreviousPassage(t *testing.T) {
	handler, closeFn, err := newServer(testServerConfig(t))
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	defer closeFn()

	first := mustReadReader(t, handler, "/api/reader")
	if len(first.Passages) < 3 {
		t.Fatalf("need at least three passages, got %d", len(first.Passages))
	}

	thirdPassageID := first.Passages[2].PassageID
	third := mustReadReader(t, handler, "/api/reader?passage="+thirdPassageID)
	if third.Passage.PassageID != thirdPassageID {
		t.Fatalf("got passage %q, want %q", third.Passage.PassageID, thirdPassageID)
	}
	index := -1
	for i, item := range third.Passages {
		if item.PassageID == third.Passage.PassageID {
			index = i
			break
		}
	}
	if index != 2 {
		t.Fatalf("got canonical index %d, want 2", index)
	}
	if third.Passages[index-1].PassageID != first.Passages[1].PassageID {
		t.Fatalf("got previous passage %q, want %q", third.Passages[index-1].PassageID, first.Passages[1].PassageID)
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
	cfg := testServerConfig(t)
	handler, closeFn, err := newServer(cfg)
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
	if response.Question.Status != string(mvp.QuestionStatusAnswered) {
		t.Fatalf("got question status %q, want %q", response.Question.Status, mvp.QuestionStatusAnswered)
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

	storage, err := mvp.OpenSQLiteWithMigrations(cfg.DBPath, mvp.SystemClock{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer storage.Close()
	userID, _ := mvp.NewUserID(defaultUserID)
	opinionID, _ := mvp.NewOpinionID(reader.Opinion.OpinionID)
	answers, err := storage.LoadAnswers(userID, opinionID)
	if err != nil {
		t.Fatalf("load saved answers: %v", err)
	}
	if len(answers) != 1 {
		t.Fatalf("got %d saved answers, want 1", len(answers))
	}
}

func TestReaderAPIIncludesRepairState(t *testing.T) {
	handler, closeFn, err := newServer(testServerConfig(t))
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	defer closeFn()

	response := mustReadReader(t, handler, "/api/reader")
	if response.Repair.CanMergeNext == false && response.Repair.CanMergePrevious == false && response.Repair.CanSplitSentence == false {
		t.Fatal("expected at least one repair action")
	}
}

func TestRepairDataExposesRemoveHeaderWhenIssuePresent(t *testing.T) {
	opinionID, _ := mvp.NewOpinionID("24-777_9ol1")
	sectionID, _ := mvp.NewSectionID("syllabus")
	passageID, _ := mvp.NewPassageID("header-passage")
	passage, err := mvp.NewPassage(
		passageID,
		opinionID,
		sectionID,
		0,
		0,
		2,
		2,
		"Text 2 URIAS-ORELLANA v. BONDI Syllabus more text.",
		nil,
		true,
	)
	if err != nil {
		t.Fatalf("new passage: %v", err)
	}
	dto := repairData(nil, opinionID, "demo", []mvp.Passage{passage}, passage)
	if !dto.CanRemoveHeader {
		t.Fatal("expected removeHeader to be available")
	}
}

func TestRepairApplyRemoveHeaderRemovesRunningHeader(t *testing.T) {
	app, closeFn, err := newSyntheticRepairServer(t, "Held: The INA requires application of the substantial-evidence standard 2 URIAS-ORELLANA v. BONDI Syllabus to the agency's determination whether a given set of undisputed facts rises to the level of persecution under §1101(a)(42)(A). Pp. 5-13.")
	if err != nil {
		t.Fatalf("new synthetic repair server: %v", err)
	}
	defer closeFn()

	request := httptest.NewRequest(http.MethodGet, "/api/reader?panel=repair", nil)
	recorder := httptest.NewRecorder()
	app.handleReader(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var reader readerResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &reader); err != nil {
		t.Fatalf("decode reader response: %v", err)
	}
	if !reader.Repair.CanRemoveHeader {
		t.Fatal("expected removeHeader to be available")
	}
	if !strings.Contains(reader.Passage.Text, "URIAS-ORELLANA v. BONDI Syllabus") {
		t.Fatalf("expected running header artifact in passage text, got %q", reader.Passage.Text)
	}

	applyBody, err := json.Marshal(repairRequest{
		UserID:    defaultUserID,
		OpinionID: reader.Opinion.OpinionID,
		PassageID: reader.Passage.PassageID,
		Operation: "removeHeader",
	})
	if err != nil {
		t.Fatalf("marshal apply payload: %v", err)
	}

	applyRequest := httptest.NewRequest(http.MethodPost, "/api/repair/apply", bytes.NewReader(applyBody))
	applyRequest.Header.Set("Content-Type", "application/json")
	applyRecorder := httptest.NewRecorder()
	app.handleRepairApply(applyRecorder, applyRequest)
	if applyRecorder.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d body=%s", applyRecorder.Code, http.StatusOK, applyRecorder.Body.String())
	}

	var applyResponse repairResponse
	if err := json.Unmarshal(applyRecorder.Body.Bytes(), &applyResponse); err != nil {
		t.Fatalf("decode apply response: %v", err)
	}
	repairedRequest := httptest.NewRequest(http.MethodGet, "/api/reader?passage="+applyResponse.PassageID+"&panel=repair", nil)
	repairedRecorder := httptest.NewRecorder()
	app.handleReader(repairedRecorder, repairedRequest)
	if repairedRecorder.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d body=%s", repairedRecorder.Code, http.StatusOK, repairedRecorder.Body.String())
	}

	var repaired readerResponse
	if err := json.Unmarshal(repairedRecorder.Body.Bytes(), &repaired); err != nil {
		t.Fatalf("decode repaired response: %v", err)
	}
	if strings.Contains(repaired.Passage.Text, "URIAS-ORELLANA v. BONDI Syllabus") {
		t.Fatalf("expected running header to be removed, got %q", repaired.Passage.Text)
	}
	if !strings.Contains(repaired.Passage.Text, "Held: The INA requires application of the substantial-evidence standard") {
		t.Fatalf("expected repaired passage to keep surrounding source text, got %q", repaired.Passage.Text)
	}
	if len(repaired.Repair.History) == 0 {
		t.Fatal("expected persisted repair history after removeHeader")
	}
	if repaired.Repair.History[len(repaired.Repair.History)-1].OperationKind != string(mvp.AdminPassageOperationRemoveRunningHeader) {
		t.Fatalf("got operation %q want %q", repaired.Repair.History[len(repaired.Repair.History)-1].OperationKind, mvp.AdminPassageOperationRemoveRunningHeader)
	}
}

func TestRepairApplyAndUndoEndpoints(t *testing.T) {
	handler, closeFn, err := newServer(testServerConfig(t))
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	defer closeFn()

	reader := mustReadReader(t, handler, "/api/reader")
	applyBody, err := json.Marshal(repairRequest{
		UserID:    defaultUserID,
		OpinionID: reader.Opinion.OpinionID,
		PassageID: reader.Passage.PassageID,
		Operation: "mergeNext",
	})
	if err != nil {
		t.Fatalf("marshal apply payload: %v", err)
	}
	applyRequest := httptest.NewRequest(http.MethodPost, "/api/repair/apply", bytes.NewReader(applyBody))
	applyRequest.Header.Set("Content-Type", "application/json")
	applyRecorder := httptest.NewRecorder()
	handler.ServeHTTP(applyRecorder, applyRequest)
	if applyRecorder.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d body=%s", applyRecorder.Code, http.StatusOK, applyRecorder.Body.String())
	}

	var applyResponse repairResponse
	if err := json.Unmarshal(applyRecorder.Body.Bytes(), &applyResponse); err != nil {
		t.Fatalf("decode apply response: %v", err)
	}
	if applyResponse.Revision != 1 {
		t.Fatalf("got revision %d want 1", applyResponse.Revision)
	}

	repaired := mustReadReader(t, handler, "/api/reader?panel=repair")
	if len(repaired.Repair.History) == 0 {
		t.Fatal("expected repair history after apply")
	}

	undoBody, err := json.Marshal(repairUndoRequest{
		UserID:    defaultUserID,
		OpinionID: reader.Opinion.OpinionID,
	})
	if err != nil {
		t.Fatalf("marshal undo payload: %v", err)
	}
	undoRequest := httptest.NewRequest(http.MethodPost, "/api/repair/undo", bytes.NewReader(undoBody))
	undoRequest.Header.Set("Content-Type", "application/json")
	undoRecorder := httptest.NewRecorder()
	handler.ServeHTTP(undoRecorder, undoRequest)
	if undoRecorder.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d body=%s", undoRecorder.Code, http.StatusOK, undoRecorder.Body.String())
	}

	var undoResponse repairResponse
	if err := json.Unmarshal(undoRecorder.Body.Bytes(), &undoResponse); err != nil {
		t.Fatalf("decode undo response: %v", err)
	}
	if undoResponse.PassageID == "" {
		t.Fatal("expected passage id after undo")
	}
}

func TestRepairApplyUndoApplyDoesNotCollideAuditRevision(t *testing.T) {
	handler, closeFn, err := newServer(testServerConfig(t))
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	defer closeFn()

	reader := mustReadReader(t, handler, "/api/reader")
	if len(reader.Passages) < 3 {
		t.Fatalf("need at least three passages, got %d", len(reader.Passages))
	}
	targetPassageID := reader.Passages[2].PassageID

	apply := func() {
		t.Helper()
		body, err := json.Marshal(repairRequest{
			UserID:    defaultUserID,
			OpinionID: reader.Opinion.OpinionID,
			PassageID: targetPassageID,
			Operation: "mergePrevious",
		})
		if err != nil {
			t.Fatalf("marshal apply payload: %v", err)
		}
		request := httptest.NewRequest(http.MethodPost, "/api/repair/apply", bytes.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("got status %d, want %d body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
		}
	}
	undo := func() {
		t.Helper()
		body, err := json.Marshal(repairUndoRequest{
			UserID:    defaultUserID,
			OpinionID: reader.Opinion.OpinionID,
		})
		if err != nil {
			t.Fatalf("marshal undo payload: %v", err)
		}
		request := httptest.NewRequest(http.MethodPost, "/api/repair/undo", bytes.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("got status %d, want %d body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
		}
	}

	apply()
	undo()
	apply()

	repaired := mustReadReader(t, handler, "/api/reader?panel=repair")
	if len(repaired.Repair.History) == 0 {
		t.Fatal("expected repair history after apply/undo/apply")
	}
}

func TestRepairPanelHistoryScopesToFocusedPassage(t *testing.T) {
	app, closeFn, err := newSyntheticMultiPassageRepairServer(t)
	if err != nil {
		t.Fatalf("new synthetic multi-passage repair server: %v", err)
	}
	defer closeFn()

	splitBody, err := json.Marshal(repairRequest{
		UserID:    defaultUserID,
		OpinionID: string(app.opinionID),
		PassageID: "syllabus-25",
		Operation: "splitSentence",
	})
	if err != nil {
		t.Fatalf("marshal split payload: %v", err)
	}
	splitRequest := httptest.NewRequest(http.MethodPost, "/api/repair/apply", bytes.NewReader(splitBody))
	splitRequest.Header.Set("Content-Type", "application/json")
	splitRecorder := httptest.NewRecorder()
	app.handleRepairApply(splitRecorder, splitRequest)
	if splitRecorder.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d body=%s", splitRecorder.Code, http.StatusOK, splitRecorder.Body.String())
	}

	removeBody, err := json.Marshal(repairRequest{
		UserID:    defaultUserID,
		OpinionID: string(app.opinionID),
		PassageID: "syllabus-27",
		Operation: "removeHeader",
	})
	if err != nil {
		t.Fatalf("marshal remove payload: %v", err)
	}
	removeRequest := httptest.NewRequest(http.MethodPost, "/api/repair/apply", bytes.NewReader(removeBody))
	removeRequest.Header.Set("Content-Type", "application/json")
	removeRecorder := httptest.NewRecorder()
	app.handleRepairApply(removeRecorder, removeRequest)
	if removeRecorder.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d body=%s", removeRecorder.Code, http.StatusOK, removeRecorder.Body.String())
	}

	syllabus27 := mustReadReaderFromServer(t, app, "/api/reader?passage=syllabus-27&panel=repair")
	if len(syllabus27.Repair.History) != 1 {
		t.Fatalf("got %d history entries for syllabus-27, want 1", len(syllabus27.Repair.History))
	}
	if syllabus27.Repair.History[0].TargetPassage != "syllabus-27" {
		t.Fatalf("got history target %q, want syllabus-27", syllabus27.Repair.History[0].TargetPassage)
	}
	if syllabus27.Repair.History[0].OperationKind != string(mvp.AdminPassageOperationRemoveRunningHeader) {
		t.Fatalf("got operation %q, want %q", syllabus27.Repair.History[0].OperationKind, mvp.AdminPassageOperationRemoveRunningHeader)
	}

	syllabus25 := mustReadReaderFromServer(t, app, "/api/reader?passage=syllabus-25&panel=repair")
	if len(syllabus25.Repair.History) != 1 {
		t.Fatalf("got %d history entries for syllabus-25, want 1", len(syllabus25.Repair.History))
	}
	if syllabus25.Repair.History[0].TargetPassage != "syllabus-25" {
		t.Fatalf("got history target %q, want syllabus-25", syllabus25.Repair.History[0].TargetPassage)
	}
	if syllabus25.Repair.History[0].OperationKind != string(mvp.AdminPassageOperationSplitAtSentence) {
		t.Fatalf("got operation %q, want %q", syllabus25.Repair.History[0].OperationKind, mvp.AdminPassageOperationSplitAtSentence)
	}
}

func TestRepairApplyMergePreviousFocusesMergedPassage(t *testing.T) {
	handler, closeFn, err := newServer(testServerConfig(t))
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	defer closeFn()

	reader := mustReadReader(t, handler, "/api/reader")
	if len(reader.Passages) < 3 {
		t.Fatalf("need at least three passages, got %d", len(reader.Passages))
	}

	thirdPassageID := reader.Passages[2].PassageID
	secondPassageID := reader.Passages[1].PassageID
	applyBody, err := json.Marshal(repairRequest{
		UserID:    defaultUserID,
		OpinionID: reader.Opinion.OpinionID,
		PassageID: thirdPassageID,
		Operation: "mergePrevious",
	})
	if err != nil {
		t.Fatalf("marshal apply payload: %v", err)
	}
	applyRequest := httptest.NewRequest(http.MethodPost, "/api/repair/apply", bytes.NewReader(applyBody))
	applyRequest.Header.Set("Content-Type", "application/json")
	applyRecorder := httptest.NewRecorder()
	handler.ServeHTTP(applyRecorder, applyRequest)
	if applyRecorder.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d body=%s", applyRecorder.Code, http.StatusOK, applyRecorder.Body.String())
	}

	var applyResponse repairResponse
	if err := json.Unmarshal(applyRecorder.Body.Bytes(), &applyResponse); err != nil {
		t.Fatalf("decode apply response: %v", err)
	}
	if applyResponse.PassageID != secondPassageID {
		t.Fatalf("got focus passage %q want %q", applyResponse.PassageID, secondPassageID)
	}
}

func TestWriteErrorLogsServerFailures(t *testing.T) {
	var logs bytes.Buffer
	app := &server{logger: log.New(&logs, "", 0)}
	request := httptest.NewRequest(http.MethodPost, "/api/repair/apply", nil)
	recorder := httptest.NewRecorder()

	app.writeError(recorder, request, http.StatusBadRequest, errors.New("split boundary is invalid"))

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("got status %d want %d", recorder.Code, http.StatusBadRequest)
	}
	if !strings.Contains(logs.String(), "path=/api/repair/apply") {
		t.Fatalf("expected request path in logs: %s", logs.String())
	}
	if !strings.Contains(logs.String(), "split boundary is invalid") {
		t.Fatalf("expected error text in logs: %s", logs.String())
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

func findRepairPassageByIssue(t *testing.T, handler http.Handler, kind mvp.PassageIssueKind) readerResponse {
	t.Helper()

	reader := mustReadReader(t, handler, "/api/reader")
	for _, item := range reader.Passages {
		candidate := mustReadReader(t, handler, "/api/reader?passage="+item.PassageID+"&panel=repair")
		for _, issue := range candidate.Repair.Issues {
			if issue.Kind == string(kind) {
				return candidate
			}
		}
	}
	t.Fatalf("no passage found for repair issue %q", kind)
	return reader
}

func newSyntheticRepairServer(t *testing.T, passageText string) (*server, func(), error) {
	t.Helper()

	clock := mvp.FixedClock{}
	storage, _, err := mvp.NewTempSQLite(clock)
	if err != nil {
		return nil, nil, err
	}

	cleanup := func() {
		_ = storage.Close()
	}

	opinionID, err := mvp.NewOpinionID("24-777_9ol1")
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	sectionID, err := mvp.NewSectionID("syllabus")
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	meta, err := mvp.NewMeta("Urias-Orellana v. Bondi", "24-777", "2026-03-17", "October Term 2025", nil)
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	section, err := mvp.NewSection(opinionID, sectionID, mvp.SectionKindSyllabus, "Syllabus", nil, 2, 2, mvp.Text(passageText))
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	opinion, err := mvp.NewOpinion(opinionID, meta, []mvp.Section{section}, mvp.Text(passageText))
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	if _, err := storage.SaveOpinion(opinion); err != nil {
		cleanup()
		return nil, nil, err
	}
	passageID, err := mvp.NewPassageID("header-passage")
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	passage, err := mvp.NewPassage(passageID, opinionID, sectionID, 0, 0, 2, 2, mvp.Text(passageText), nil, true)
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	if _, err := storage.SavePassages([]mvp.PassageRecord{passage}); err != nil {
		cleanup()
		return nil, nil, err
	}
	userID, err := mvp.NewUserID(defaultUserID)
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	if _, err := mvp.OpenPassage(userID, passageID, storage); err != nil {
		cleanup()
		return nil, nil, err
	}

	app := &server{
		storage:   storage,
		userID:    userID,
		opinionID: opinionID,
		logger:    log.New(&bytes.Buffer{}, "sprout-web ", log.LstdFlags),
	}
	return app, cleanup, nil
}

func newSyntheticMultiPassageRepairServer(t *testing.T) (*server, func(), error) {
	t.Helper()

	clock := mvp.FixedClock{}
	storage, _, err := mvp.NewTempSQLite(clock)
	if err != nil {
		return nil, nil, err
	}

	cleanup := func() {
		_ = storage.Close()
	}

	opinionID, err := mvp.NewOpinionID("24-777_9ol1")
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	sectionID, err := mvp.NewSectionID("syllabus")
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	meta, err := mvp.NewMeta("Urias-Orellana v. Bondi", "24-777", "2026-03-17", "October Term 2025", nil)
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	fullText := "First passage. Second sentence. Middle passage. Held: The INA requires application of the substantial-evidence standard 2 URIAS-ORELLANA v. BONDI Syllabus to the agency's determination whether a given set of undisputed facts rises to the level of persecution under §1101(a)(42)(A). Pp. 5-13."
	section, err := mvp.NewSection(opinionID, sectionID, mvp.SectionKindSyllabus, "Syllabus", nil, 2, 2, mvp.Text(fullText))
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	opinion, err := mvp.NewOpinion(opinionID, meta, []mvp.Section{section}, mvp.Text(fullText))
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	if _, err := storage.SaveOpinion(opinion); err != nil {
		cleanup()
		return nil, nil, err
	}

	firstID, _ := mvp.NewPassageID("syllabus-25")
	secondID, _ := mvp.NewPassageID("syllabus-26")
	thirdID, _ := mvp.NewPassageID("syllabus-27")
	first, err := mvp.NewPassage(firstID, opinionID, sectionID, 0, 1, 2, 2, "First passage. Second sentence.", nil, true)
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	second, err := mvp.NewPassage(secondID, opinionID, sectionID, 2, 2, 2, 2, "Middle passage.", nil, true)
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	third, err := mvp.NewPassage(thirdID, opinionID, sectionID, 3, 3, 2, 2, "Held: The INA requires application of the substantial-evidence standard 2 URIAS-ORELLANA v. BONDI Syllabus to the agency's determination whether a given set of undisputed facts rises to the level of persecution under §1101(a)(42)(A). Pp. 5-13.", nil, true)
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	if _, err := storage.SavePassages([]mvp.PassageRecord{first, second, third}); err != nil {
		cleanup()
		return nil, nil, err
	}

	userID, err := mvp.NewUserID(defaultUserID)
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	if _, err := mvp.OpenPassage(userID, firstID, storage); err != nil {
		cleanup()
		return nil, nil, err
	}

	app := &server{
		storage:   storage,
		userID:    userID,
		opinionID: opinionID,
		logger:    log.New(&bytes.Buffer{}, "sprout-web ", log.LstdFlags),
	}
	return app, cleanup, nil
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

func mustReadReaderFromServer(t *testing.T, app *server, path string) readerResponse {
	t.Helper()

	request := httptest.NewRequest(http.MethodGet, path, nil)
	recorder := httptest.NewRecorder()
	app.handleReader(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	var response readerResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return response
}
