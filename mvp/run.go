package mvp

import (
	"context"
	"net/http"
)

func RunMVP(ctx context.Context, userID UserID, input UserInput, client *http.Client, storage Storage, clock Clock, model Model) (ReadingState, error) {
	opinionURL, err := EnterURL(string(input.URL))
	if err != nil {
		return ReadingState{}, err
	}
	opinionID, err := MakeOpinionID(opinionURL)
	if err != nil {
		return ReadingState{}, err
	}

	bytes, err := FetchPDF(ctx, client, opinionURL)
	if err != nil {
		return ReadingState{}, err
	}
	raw, err := MakeRawPDF(opinionID, opinionURL, bytes, clock.Now())
	if err != nil {
		return ReadingState{}, err
	}

	firstPassageID, err := IngestRawPDF(userID, raw, storage, model)
	if err != nil {
		return ReadingState{}, err
	}
	return OpenPassage(userID, firstPassageID, storage)
}

func IngestRawPDF(userID UserID, raw RawPDF, storage Storage, model Model) (PassageID, error) {
	var firstPassageID PassageID
	var err error
	opinionID := raw.OpinionID
	if transactional, ok := storage.(TransactionalStorage); ok {
		err = transactional.InTx(func(txStorage Storage) error {
			first, txErr := runMVPWrites(opinionID, raw, userID, txStorage, model)
			if txErr != nil {
				return txErr
			}
			firstPassageID = first
			return nil
		})
		if err != nil {
			return "", err
		}
	} else {
		firstPassageID, err = runMVPWrites(opinionID, raw, userID, storage, model)
		if err != nil {
			return "", err
		}
	}
	return firstPassageID, nil
}

func runMVPWrites(opinionID OpinionID, raw RawPDF, userID UserID, storage Storage, model Model) (PassageID, error) {
	if _, err := StorePDF(storage, raw); err != nil {
		return "", err
	}

	parsed, err := ParsePDF(raw)
	if err != nil {
		return "", err
	}
	meta, err := ExtractMeta(parsed)
	if err != nil {
		return "", err
	}
	sections, err := GuessSections(model, parsed)
	if err != nil {
		return "", err
	}
	opinion, err := BuildOpinion(opinionID, meta, sections, parsed)
	if err != nil {
		return "", err
	}
	if _, err := StoreOpinion(storage, opinion); err != nil {
		return "", err
	}

	chunkPolicy := DefaultChunkPolicy()
	screenPolicy := DefaultScreenPolicy()

	passages, err := ChunkSections(chunkPolicy, sections)
	if err != nil {
		return "", err
	}
	passages, err = AttachCitations(passages)
	if err != nil {
		return "", err
	}
	passages, err = RepairPassages(screenPolicy, passages)
	if err != nil {
		return "", err
	}
	passages, err = StorePassages(storage, passages)
	if err != nil {
		return "", err
	}

	queue, err := BuildQueue(userID, opinion, passages)
	if err != nil {
		return "", err
	}
	next, err := NextPassage(queue)
	if err != nil {
		return "", err
	}
	return *next, nil
}
