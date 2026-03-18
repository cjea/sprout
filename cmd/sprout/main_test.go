package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"sprout/mvp"
)

const fixtureURL = "https://www.supremecourt.gov/opinions/25pdf/24-777_9ol1.pdf"

func TestInitIngestAndReadFlow(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "sprout.db")

	runOK(t, "init-db", "--db", dbPath)

	fixturePath := mustFixturePath(t)
	ingestOutput := runOK(t,
		"ingest-file",
		"--db", dbPath,
		"--user", "demo",
		"--file", fixturePath,
		"--url", fixtureURL,
	)
	if !strings.Contains(ingestOutput, "opinion_id=24-777_9ol1") {
		t.Fatalf("ingest output missing opinion id: %s", ingestOutput)
	}

	showOpinionOutput := runOK(t,
		"show-opinion",
		"--db", dbPath,
		"--opinion-id", "24-777_9ol1",
	)
	if !strings.Contains(showOpinionOutput, "case_name=Urias-Orellana v. Bondi") {
		t.Fatalf("show-opinion output missing case name: %s", showOpinionOutput)
	}

	listOutput := runOK(t,
		"list-passages",
		"--db", dbPath,
		"--opinion-id", "24-777_9ol1",
	)
	if !strings.Contains(listOutput, "passage_id=") {
		t.Fatalf("list-passages output missing passages: %s", listOutput)
	}

	readOutput := runOK(t,
		"read",
		"--db", dbPath,
		"--user", "demo",
		"--opinion-id", "24-777_9ol1",
	)
	if !strings.Contains(readOutput, "case_name=Urias-Orellana v. Bondi") {
		t.Fatalf("read output missing case name: %s", readOutput)
	}

	progressOutput := runOK(t,
		"show-progress",
		"--db", dbPath,
		"--user", "demo",
		"--opinion-id", "24-777_9ol1",
	)
	if !strings.Contains(progressOutput, "current_passage_id=") {
		t.Fatalf("show-progress output missing current passage: %s", progressOutput)
	}
}

func TestAskCommand(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "sprout.db")
	runOK(t, "init-db", "--db", dbPath)
	runOK(t,
		"ingest-file",
		"--db", dbPath,
		"--user", "demo",
		"--file", mustFixturePath(t),
		"--url", fixtureURL,
	)

	storage, err := openStorage(dbPath)
	if err != nil {
		t.Fatalf("open storage: %v", err)
	}
	defer storage.Close()

	passages, err := listPassages(storage, mvp.OpinionID("24-777_9ol1"))
	if err != nil {
		t.Fatalf("list passages: %v", err)
	}
	if len(passages) == 0 {
		t.Fatal("expected passages")
	}

	passage := passages[0]
	end := 32
	if len(passage.Text) < end {
		end = len(passage.Text)
	}
	if end <= 1 {
		t.Fatalf("passage too short for ask test: %q", passage.Text)
	}

	askOutput := runOK(t,
		"ask",
		"--db", dbPath,
		"--user", "demo",
		"--passage-id", string(passage.PassageID),
		"--start", "0",
		"--end", strconv.Itoa(end),
		"--question", "What does this sentence mean?",
	)
	if !strings.Contains(askOutput, "answer=") {
		t.Fatalf("ask output missing answer: %s", askOutput)
	}
	if !strings.Contains(askOutput, "question_id=") {
		t.Fatalf("ask output missing question id: %s", askOutput)
	}
}

func TestRepairCaseCommand(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "sprout.db")
	runOK(t, "init-db", "--db", dbPath)
	runOK(t,
		"ingest-file",
		"--db", dbPath,
		"--user", "demo",
		"--file", mustFixturePath(t),
		"--url", fixtureURL,
	)

	storage, err := openStorage(dbPath)
	if err != nil {
		t.Fatalf("open storage: %v", err)
	}
	defer storage.Close()

	passages, err := listPassages(storage, mvp.OpinionID("24-777_9ol1"))
	if err != nil {
		t.Fatalf("list passages: %v", err)
	}
	if len(passages) < 2 {
		t.Fatalf("expected at least two passages")
	}

	output := runOK(t,
		"repair-case",
		"--db", dbPath,
		"--opinion-id", "24-777_9ol1",
		"--passage-id", string(passages[1].PassageID),
	)
	if !strings.Contains(output, "stage=apply_operation") {
		t.Fatalf("repair-case output missing stage: %s", output)
	}
	if !strings.Contains(output, "current_passage_id="+string(passages[1].PassageID)) {
		t.Fatalf("repair-case output missing current passage: %s", output)
	}
	if !strings.Contains(output, "previous_passage_id=") {
		t.Fatalf("repair-case output missing previous passage context: %s", output)
	}
	if !strings.Contains(output, "next_passage_id=") {
		t.Fatalf("repair-case output missing next passage context: %s", output)
	}
}

func TestRepairIssueApplyHistoryAndUndoCommands(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "sprout.db")
	runOK(t, "init-db", "--db", dbPath)
	runOK(t,
		"ingest-file",
		"--db", dbPath,
		"--user", "demo",
		"--file", mustFixturePath(t),
		"--url", fixtureURL,
	)

	storage, err := openStorage(dbPath)
	if err != nil {
		t.Fatalf("open storage: %v", err)
	}
	defer storage.Close()

	passages, err := listPassages(storage, mvp.OpinionID("24-777_9ol1"))
	if err != nil {
		t.Fatalf("list passages: %v", err)
	}
	if len(passages) < 2 {
		t.Fatalf("expected at least two passages")
	}
	originalCount := len(passages)

	issuesOutput := runOK(t,
		"repair-issues",
		"--db", dbPath,
		"--opinion-id", "24-777_9ol1",
		"--passage-id", string(passages[0].PassageID),
	)
	if !strings.Contains(issuesOutput, "issue_count=") {
		t.Fatalf("repair-issues output missing issue_count: %s", issuesOutput)
	}

	applyOutput := runOK(t,
		"repair-apply",
		"--db", dbPath,
		"--opinion-id", "24-777_9ol1",
		"--passage-id", string(passages[0].PassageID),
		"--op", "merge_with_next",
	)
	if !strings.Contains(applyOutput, "operation=merge_with_next") {
		t.Fatalf("repair-apply output missing operation: %s", applyOutput)
	}

	updatedPassages, err := listPassages(storage, mvp.OpinionID("24-777_9ol1"))
	if err != nil {
		t.Fatalf("list updated passages: %v", err)
	}
	if len(updatedPassages) != originalCount-1 {
		t.Fatalf("got %d passages want %d", len(updatedPassages), originalCount-1)
	}

	historyOutput := runOK(t,
		"repair-history",
		"--db", dbPath,
		"--opinion-id", "24-777_9ol1",
	)
	if !strings.Contains(historyOutput, "operation=merge_with_next") {
		t.Fatalf("repair-history output missing operation: %s", historyOutput)
	}

	undoOutput := runOK(t,
		"repair-undo",
		"--db", dbPath,
		"--opinion-id", "24-777_9ol1",
	)
	if !strings.Contains(undoOutput, "revision=0") {
		t.Fatalf("repair-undo output missing revision reset: %s", undoOutput)
	}

	restoredPassages, err := listPassages(storage, mvp.OpinionID("24-777_9ol1"))
	if err != nil {
		t.Fatalf("list restored passages: %v", err)
	}
	if len(restoredPassages) != originalCount {
		t.Fatalf("got %d passages want %d", len(restoredPassages), originalCount)
	}
}

func runOK(t *testing.T, args ...string) string {
	t.Helper()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run(args, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run failed: code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	return stdout.String()
}

func mustFixturePath(t *testing.T) string {
	t.Helper()

	root, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	path := filepath.Join(root, "..", "..", "fixtures", "scotus", "24-777_9ol1.pdf")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("stat fixture: %v", err)
	}
	return path
}
