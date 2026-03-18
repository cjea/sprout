CREATE TABLE IF NOT EXISTS answers (
    question_id TEXT PRIMARY KEY,
    answer TEXT NOT NULL,
    evidence_json TEXT NOT NULL,
    caveats_json TEXT NOT NULL,
    generated_at TEXT NOT NULL,
    model_name TEXT NOT NULL,
    FOREIGN KEY (question_id) REFERENCES questions(question_id) ON DELETE CASCADE
);
