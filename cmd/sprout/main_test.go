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
