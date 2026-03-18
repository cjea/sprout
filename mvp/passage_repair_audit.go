package mvp

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

var ErrPassageRepairAuditEmpty = errors.New("passage repair audit is empty")

type PassageRepairAuditEntry struct {
	SessionID          string
	Revision           int
	OpinionID          OpinionID
	ActorID            string
	Source             string
	OperationKind      AdminPassageOperationKind
	TargetPassageID    PassageID
	SplitAfterSentence *SentenceNo
	Before             PassageRepairSnapshot
	After              PassageRepairSnapshot
	CreatedAt          Timestamp
}

type PassageRepairAuditStore interface {
	LoadLatestPassageRepairSession(context.Context, OpinionID, string) (*PassageRepairSession, error)
	ListPassageRepairAudit(context.Context, OpinionID, string) ([]PassageRepairAuditEntry, error)
	RecordPassageRepairOperation(context.Context, PassageRepairAuditEntry) error
}

func LoadOrStartAuditedPassageRepairSession(ctx context.Context, auditStore PassageRepairAuditStore, opinionID OpinionID, actorID string, source string, passages []Passage) (*PassageRepairSession, error) {
	session, err := auditStore.LoadLatestPassageRepairSession(ctx, opinionID, actorID)
	if err == nil {
		return reconcileAuditedPassageRepairSession(session, passages)
	}
	if !errors.Is(err, ErrNotFound) && !errors.Is(err, ErrPassageRepairAuditEmpty) {
		return nil, err
	}
	snapshot, err := NewPassageRepairSnapshot(defaultRepairSessionID(opinionID, actorID), 0, opinionID, passages, nil)
	if err != nil {
		return nil, err
	}
	return NewPassageRepairSession(snapshot)
}

func reconcileAuditedPassageRepairSession(session *PassageRepairSession, passages []Passage) (*PassageRepairSession, error) {
	if session == nil {
		return nil, ErrNotFound
	}
	if samePassageIDs(session.Current.Passages, passages) {
		return session, nil
	}
	snapshot, err := NewPassageRepairSnapshot(session.Current.SessionID, session.Current.Revision, session.Current.OpinionID, passages, nil)
	if err != nil {
		return nil, err
	}
	return &PassageRepairSession{
		Current: snapshot,
		History: append([]PassageRepairHistoryEntry(nil), session.History...),
	}, nil
}

func samePassageIDs(left, right []Passage) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index].PassageID != right[index].PassageID {
			return false
		}
	}
	return true
}

func RecordAuditedPassageRepairOperation(ctx context.Context, auditStore PassageRepairAuditStore, actorID string, source string, operation AdminPassageOperation, before PassageRepairSnapshot, after PassageRepairSnapshot, createdAt Timestamp) error {
	entry := PassageRepairAuditEntry{
		SessionID:          after.SessionID,
		Revision:           after.Revision,
		OpinionID:          after.OpinionID,
		ActorID:            actorID,
		Source:             source,
		OperationKind:      operation.Kind,
		TargetPassageID:    operation.Target.PassageIDs[0],
		SplitAfterSentence: operation.SplitAfterSentence,
		Before:             before,
		After:              after,
		CreatedAt:          createdAt,
	}
	return auditStore.RecordPassageRepairOperation(ctx, entry)
}

func (s *SQLiteStorage) LoadLatestPassageRepairSession(ctx context.Context, opinionID OpinionID, actorID string) (*PassageRepairSession, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT after_snapshot_json
		FROM repair_audit
		WHERE opinion_id = ? AND actor_id = ?
		ORDER BY revision DESC
		LIMIT 1
	`, opinionID, actorID)
	var snapshotJSON string
	if err := row.Scan(&snapshotJSON); errors.Is(err, sql.ErrNoRows) {
		return nil, ErrPassageRepairAuditEmpty
	} else if err != nil {
		return nil, err
	}
	var snapshot PassageRepairSnapshot
	if err := json.Unmarshal([]byte(snapshotJSON), &snapshot); err != nil {
		return nil, err
	}
	return NewPassageRepairSession(snapshot)
}

func (s *SQLiteStorage) ListPassageRepairAudit(ctx context.Context, opinionID OpinionID, actorID string) ([]PassageRepairAuditEntry, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT session_id, revision, opinion_id, actor_id, source, operation_kind, target_passage_id, split_after_sentence, before_snapshot_json, after_snapshot_json, created_at
		FROM repair_audit
		WHERE opinion_id = ? AND actor_id = ?
		ORDER BY revision ASC
	`, opinionID, actorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := make([]PassageRepairAuditEntry, 0)
	for rows.Next() {
		var entry PassageRepairAuditEntry
		var split sql.NullInt64
		var beforeJSON string
		var afterJSON string
		var createdAt string
		if err := rows.Scan(&entry.SessionID, &entry.Revision, &entry.OpinionID, &entry.ActorID, &entry.Source, &entry.OperationKind, &entry.TargetPassageID, &split, &beforeJSON, &afterJSON, &createdAt); err != nil {
			return nil, err
		}
		if split.Valid {
			value := SentenceNo(split.Int64)
			entry.SplitAfterSentence = &value
		}
		if err := json.Unmarshal([]byte(beforeJSON), &entry.Before); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(afterJSON), &entry.After); err != nil {
			return nil, err
		}
		parsed, err := time.Parse(time.RFC3339Nano, createdAt)
		if err != nil {
			return nil, err
		}
		entry.CreatedAt = parsed
		entries = append(entries, entry)
	}
	if len(entries) == 0 {
		return nil, ErrPassageRepairAuditEmpty
	}
	return entries, rows.Err()
}

func (s *SQLiteStorage) RecordPassageRepairOperation(ctx context.Context, entry PassageRepairAuditEntry) error {
	revision, err := s.nextRepairAuditRevision(ctx, entry.SessionID)
	if err != nil {
		return err
	}
	entry.Revision = revision
	beforeJSON, err := json.Marshal(entry.Before)
	if err != nil {
		return err
	}
	afterJSON, err := json.Marshal(entry.After)
	if err != nil {
		return err
	}
	var split any
	if entry.SplitAfterSentence != nil {
		split = int(*entry.SplitAfterSentence)
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO repair_audit(session_id, revision, opinion_id, actor_id, source, operation_kind, target_passage_id, split_after_sentence, before_snapshot_json, after_snapshot_json, created_at)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, entry.SessionID, entry.Revision, entry.OpinionID, entry.ActorID, entry.Source, entry.OperationKind, entry.TargetPassageID, split, string(beforeJSON), string(afterJSON), entry.CreatedAt.UTC().Format(time.RFC3339Nano))
	return err
}

func (s *SQLiteStorage) nextRepairAuditRevision(ctx context.Context, sessionID string) (int, error) {
	var currentMax sql.NullInt64
	if err := s.db.QueryRowContext(ctx, `
		SELECT MAX(revision)
		FROM repair_audit
		WHERE session_id = ?
	`, sessionID).Scan(&currentMax); err != nil {
		return 0, err
	}
	if !currentMax.Valid {
		return 1, nil
	}
	return int(currentMax.Int64) + 1, nil
}

func defaultRepairSessionID(opinionID OpinionID, actorID string) string {
	return string(opinionID) + ":" + actorID
}
