package mvp

import "testing"

func TestDetectSuspiciousSentenceBoundariesFlagsLegalFragments(t *testing.T) {
	testCases := []struct {
		name           string
		sentences      []string
		wantReason     BoundarySuspicion
		wantLeftIndex  SentenceNo
		wantRightIndex SentenceNo
	}{
		{
			name: "page reference lead",
			sentences: []string{
				"Held: Courts should review the Board's determination.",
				"Pp.",
				"5-13.",
			},
			wantReason:     BoundarySuspicionPageReferenceLead,
			wantLeftIndex:  0,
			wantRightIndex: 1,
		},
		{
			name: "page range after pp",
			sentences: []string{
				"Held: Courts should review the Board's determination.",
				"Pp.",
				"5-13.",
			},
			wantReason:     BoundarySuspicionPageRangeContinuation,
			wantLeftIndex:  1,
			wantRightIndex: 2,
		},
		{
			name: "page range after pp with en dash",
			sentences: []string{
				"Held: Courts should review the Board's determination.",
				"Pp.",
				"5–13.",
			},
			wantReason:     BoundarySuspicionPageRangeContinuation,
			wantLeftIndex:  1,
			wantRightIndex: 2,
		},
		{
			name: "pin cite after id",
			sentences: []string{
				"The court rejected that argument.",
				"Id.",
				", at 484.",
			},
			wantReason:     BoundarySuspicionPinCiteContinuation,
			wantLeftIndex:  1,
			wantRightIndex: 2,
		},
		{
			name: "statute citation split from proposition",
			sentences: []string{
				"Under the INA, the U. S. Government may grant asylum to a noncitizen if it determines that he is a refugee.",
				"8 U. S. C. §1158(b)(1)(A).",
			},
			wantReason:     BoundarySuspicionTrailingAuthority,
			wantLeftIndex:  0,
			wantRightIndex: 1,
		},
		{
			name: "case citation split from quoted proposition",
			sentences: []string{
				"Substantial evidence means only such relevant evidence as a reasonable mind might accept as adequate to support a conclusion.",
				"Biestek v. Berryhill, 587 U. S. 97, 103.",
			},
			wantReason:     BoundarySuspicionTrailingAuthority,
			wantLeftIndex:  0,
			wantRightIndex: 1,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got := DetectSuspiciousSentenceBoundaries(testCase.sentences)
			if len(got) == 0 {
				t.Fatalf("expected suspicious boundaries")
			}
			var matched bool
			for _, boundary := range got {
				if boundary.Reason == testCase.wantReason &&
					boundary.LeftIndex == testCase.wantLeftIndex &&
					boundary.RightIndex == testCase.wantRightIndex {
					matched = true
					break
				}
			}
			if !matched {
				t.Fatalf("did not find boundary %+v in %#v", testCase, got)
			}
		})
	}
}

func TestDetectSuspiciousSentenceBoundariesLeavesCleanBoundariesAlone(t *testing.T) {
	corpus := loadQualityCorpus(t)

	for _, testCase := range corpus.Cases {
		if testCase.Kind != "sentence_boundary" {
			continue
		}

		t.Run(testCase.ID, func(t *testing.T) {
			got := DetectSuspiciousSentenceBoundaries(testCase.WantSentences)
			if len(got) != 0 {
				t.Fatalf("got suspicious boundaries for clean sentences: %#v", got)
			}
		})
	}
}
