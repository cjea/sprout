CREATE TABLE IF NOT EXISTS repair_audit (
    session_id TEXT NOT NULL,
    revision INTEGER NOT NULL,
    opinion_id TEXT NOT NULL,
    actor_id TEXT NOT NULL,
    source TEXT NOT NULL,
    operation_kind TEXT NOT NULL,
    target_passage_id TEXT NOT NULL,
    split_after_sentence INTEGER,
    before_snapshot_json TEXT NOT NULL,
    after_snapshot_json TEXT NOT NULL,
    created_at TEXT NOT NULL,
    PRIMARY KEY (session_id, revision),
    FOREIGN KEY (opinion_id) REFERENCES opinions(opinion_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_repair_audit_opinion_created
    ON repair_audit(opinion_id, created_at DESC);
