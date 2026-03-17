package mvp

import "testing"

func TestMemoryStorageRoundTrip(t *testing.T) {
	storage := NewMemoryStorage()
	opinionID, _ := NewOpinionID("24-777")
	sectionID, _ := NewSectionID("majority")
	passageID, _ := NewPassageID("p-1")
	userID, _ := NewUserID("reader-1")
	questionID, _ := NewQuestionID("q-1")

	raw, _ := NewRawPDF(opinionID, URL("https://example.com/opinion.pdf"), PDFBytes("text"), Timestamp{})
	if _, err := saveRawPDF(storage, raw); err != nil {
		t.Fatalf("save raw pdf: %v", err)
	}
	if _, err := loadRawPDF(storage, opinionID); err != nil {
		t.Fatalf("load raw pdf: %v", err)
	}

	meta, _ := NewMeta("Case", "24-777", "2026-03-16", "2025", nil)
	section, _ := NewSection(opinionID, sectionID, SectionKindMajority, "Majority", nil, 1, 1, "text")
	opinion, _ := NewOpinion(opinionID, meta, []Section{section}, "text")
	if _, err := saveOpinion(storage, opinion); err != nil {
		t.Fatalf("save opinion: %v", err)
	}

	passage, _ := NewPassage(passageID, opinionID, sectionID, 0, 0, 1, 1, "text", nil, true)
	if _, err := savePassages(storage, []Passage{passage}); err != nil {
		t.Fatalf("save passages: %v", err)
	}

	progress, _ := NewProgress(userID, opinionID, &passageID, []PassageID{passageID}, nil, Timestamp{})
	if _, err := saveProgress(storage, progress); err != nil {
		t.Fatalf("save progress: %v", err)
	}

	span, _ := NewSpan(0, 4, "text")
	anchor, _ := NewAnchor(opinionID, sectionID, passageID, span)
	question, _ := NewQuestion(questionID, userID, anchor, "what?", Timestamp{}, QuestionStatusOpen)
	if _, err := saveQuestion(storage, question); err != nil {
		t.Fatalf("save question: %v", err)
	}

	if _, err := loadOpinion(storage, opinionID); err != nil {
		t.Fatalf("load opinion: %v", err)
	}
	if _, err := loadPassage(storage, passageID); err != nil {
		t.Fatalf("load passage: %v", err)
	}
	if _, err := loadProgress(storage, userID, opinionID); err != nil {
		t.Fatalf("load progress: %v", err)
	}
	questions, err := loadQuestions(storage, userID, opinionID)
	if err != nil {
		t.Fatalf("load questions: %v", err)
	}
	if len(questions) != 1 {
		t.Fatalf("got %d questions, want 1", len(questions))
	}
}
