package mvp

import (
	"errors"
	"testing"
	"time"
)

type fakeRawPDF struct{}

func (fakeRawPDF) RawPDFRecord() {}

type fakeOpinion struct{}

func (fakeOpinion) OpinionRecord() {}

type fakePassage struct{}

func (fakePassage) PassageRecord() {}

type fakeProgress struct{}

func (fakeProgress) ProgressRecord() {}

type fakeQuestion struct{}

func (fakeQuestion) QuestionRecord() {}

type fakeStorage struct{}

func (fakeStorage) SaveRawPDF(record RawPDFRecord) (RawPDFRecord, error) {
	return record, nil
}

func (fakeStorage) SaveOpinion(record OpinionRecord) (OpinionRecord, error) {
	return record, nil
}

func (fakeStorage) SavePassages(records []PassageRecord) ([]PassageRecord, error) {
	return records, nil
}

func (fakeStorage) SaveProgress(record ProgressRecord) (ProgressRecord, error) {
	return record, nil
}

func (fakeStorage) SaveQuestionRecord(record QuestionRecord) (QuestionRecord, error) {
	return record, nil
}

func (fakeStorage) LoadRawPDF(OpinionID) (RawPDFRecord, error) {
	return fakeRawPDF{}, nil
}

func (fakeStorage) LoadOpinion(OpinionID) (OpinionRecord, error) {
	return fakeOpinion{}, nil
}

func (fakeStorage) LoadPassage(PassageID) (PassageRecord, error) {
	return fakePassage{}, nil
}

func (fakeStorage) LoadProgress(UserID, OpinionID) (ProgressRecord, error) {
	return fakeProgress{}, nil
}

func (fakeStorage) LoadQuestions(UserID, OpinionID) ([]QuestionRecord, error) {
	return []QuestionRecord{fakeQuestion{}}, nil
}

func TestNewUserInput(t *testing.T) {
	tests := []struct {
		name string
		url  URL
		want error
	}{
		{name: "valid", url: URL("https://www.supremecourt.gov/opinions/25pdf/24-777_9ol1.pdf")},
		{name: "empty", url: URL(""), want: ErrEmptyURL},
		{name: "whitespace", url: URL("   "), want: ErrEmptyURL},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input, err := NewUserInput(tt.url)
			if !errors.Is(err, tt.want) {
				t.Fatalf("expected error %v, got %v", tt.want, err)
			}
			if tt.want == nil && input.URL != tt.url {
				t.Fatalf("got url %q, want %q", input.URL, tt.url)
			}
		})
	}
}

func TestNewIDs(t *testing.T) {
	tests := []struct {
		name    string
		makeID  func(string) (string, error)
		input   string
		want    string
		wantErr error
	}{
		{
			name: "opinion id trims whitespace",
			makeID: func(value string) (string, error) {
				id, err := NewOpinionID(value)
				return string(id), err
			},
			input: " 24-777 ",
			want:  "24-777",
		},
		{
			name: "user id empty",
			makeID: func(value string) (string, error) {
				id, err := NewUserID(value)
				return string(id), err
			},
			input:   "",
			wantErr: ErrEmptyUserID,
		},
		{
			name: "section id empty",
			makeID: func(value string) (string, error) {
				id, err := NewSectionID(value)
				return string(id), err
			},
			input:   " ",
			wantErr: ErrEmptySectionID,
		},
		{
			name: "passage id valid",
			makeID: func(value string) (string, error) {
				id, err := NewPassageID(value)
				return string(id), err
			},
			input: "p-1",
			want:  "p-1",
		},
		{
			name: "question id valid",
			makeID: func(value string) (string, error) {
				id, err := NewQuestionID(value)
				return string(id), err
			},
			input: "q-1",
			want:  "q-1",
		},
		{
			name: "citation id empty",
			makeID: func(value string) (string, error) {
				id, err := NewCitationID(value)
				return string(id), err
			},
			input:   "",
			wantErr: ErrEmptyCitationID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.makeID(tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected error %v, got %v", tt.wantErr, err)
			}
			if tt.wantErr == nil && got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestUserInputValidate(t *testing.T) {
	input := UserInput{}
	if !errors.Is(input.Validate(), ErrEmptyURL) {
		t.Fatalf("expected ErrEmptyURL for zero-value user input")
	}
}

func TestFixedClockNow(t *testing.T) {
	now := time.Date(2026, time.March, 16, 22, 30, 0, 0, time.UTC)
	clock := FixedClock{Time: now}

	if got := clock.Now(); !got.Equal(now) {
		t.Fatalf("got %v, want %v", got, now)
	}
}

func TestNewModel(t *testing.T) {
	tests := []struct {
		name   string
		model  string
		tokens int
		want   error
	}{
		{name: "valid", model: "gpt-5", tokens: 8000},
		{name: "empty name", model: "", tokens: 8000, want: ErrEmptyModelName},
		{name: "whitespace name", model: "   ", tokens: 8000, want: ErrEmptyModelName},
		{name: "zero tokens", model: "gpt-5", tokens: 0, want: ErrInvalidModelTokens},
		{name: "negative tokens", model: "gpt-5", tokens: -1, want: ErrInvalidModelTokens},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, err := NewModel(tt.model, tt.tokens)
			if !errors.Is(err, tt.want) {
				t.Fatalf("expected error %v, got %v", tt.want, err)
			}
			if tt.want == nil {
				if model.Name != tt.model {
					t.Fatalf("got model name %q, want %q", model.Name, tt.model)
				}
				if model.MaxContextTokens != tt.tokens {
					t.Fatalf("got max tokens %d, want %d", model.MaxContextTokens, tt.tokens)
				}
			}
		})
	}
}

func TestModelValidateZeroValue(t *testing.T) {
	var model Model
	if !errors.Is(model.Validate(), ErrEmptyModelName) {
		t.Fatalf("expected zero-value model to fail with ErrEmptyModelName")
	}
}

func TestStorageInterfaceSatisfied(t *testing.T) {
	var _ Storage = fakeStorage{}
}

func TestMarkerInterfacesSatisfied(t *testing.T) {
	var raw RawPDFRecord = fakeRawPDF{}
	var opinion OpinionRecord = fakeOpinion{}
	var passage PassageRecord = fakePassage{}
	var progress ProgressRecord = fakeProgress{}
	var question QuestionRecord = fakeQuestion{}

	if raw == nil || opinion == nil || passage == nil || progress == nil || question == nil {
		t.Fatalf("expected all fake records to satisfy their marker interfaces")
	}
}
