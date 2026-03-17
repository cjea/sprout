PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS schema_migrations (
    version TEXT PRIMARY KEY,
    applied_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS raw_pdfs (
    opinion_id TEXT PRIMARY KEY,
    source_url TEXT NOT NULL,
    bytes BLOB NOT NULL,
    fetched_at TEXT NOT NULL,
    sha256 TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS opinions (
    opinion_id TEXT PRIMARY KEY,
    case_name TEXT NOT NULL,
    docket_number TEXT NOT NULL,
    decided_on TEXT NOT NULL,
    term_label TEXT NOT NULL,
    primary_author TEXT,
    full_text TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS sections (
    opinion_id TEXT NOT NULL,
    section_id TEXT NOT NULL,
    kind TEXT NOT NULL,
    title TEXT NOT NULL,
    author TEXT,
    start_page INTEGER NOT NULL,
    end_page INTEGER NOT NULL,
    text TEXT NOT NULL,
    sort_index INTEGER NOT NULL,
    PRIMARY KEY (opinion_id, section_id),
    FOREIGN KEY (opinion_id) REFERENCES opinions(opinion_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS passages (
    passage_id TEXT PRIMARY KEY,
    opinion_id TEXT NOT NULL,
    section_id TEXT NOT NULL,
    sentence_start INTEGER NOT NULL,
    sentence_end INTEGER NOT NULL,
    page_start INTEGER NOT NULL,
    page_end INTEGER NOT NULL,
    text TEXT NOT NULL,
    fits_on_screen INTEGER NOT NULL,
    FOREIGN KEY (opinion_id) REFERENCES opinions(opinion_id) ON DELETE CASCADE,
    FOREIGN KEY (opinion_id, section_id) REFERENCES sections(opinion_id, section_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS citations (
    citation_id TEXT PRIMARY KEY,
    passage_id TEXT NOT NULL,
    kind TEXT NOT NULL,
    raw_text TEXT NOT NULL,
    normalized TEXT,
    start_offset INTEGER NOT NULL,
    end_offset INTEGER NOT NULL,
    quote TEXT NOT NULL,
    sort_index INTEGER NOT NULL,
    FOREIGN KEY (passage_id) REFERENCES passages(passage_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS progress (
    user_id TEXT NOT NULL,
    opinion_id TEXT NOT NULL,
    current_passage_id TEXT,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (user_id, opinion_id),
    FOREIGN KEY (opinion_id) REFERENCES opinions(opinion_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS completed_passages (
    user_id TEXT NOT NULL,
    opinion_id TEXT NOT NULL,
    passage_id TEXT NOT NULL,
    sort_index INTEGER NOT NULL,
    PRIMARY KEY (user_id, opinion_id, passage_id),
    FOREIGN KEY (user_id, opinion_id) REFERENCES progress(user_id, opinion_id) ON DELETE CASCADE,
    FOREIGN KEY (passage_id) REFERENCES passages(passage_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS questions (
    question_id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    opinion_id TEXT NOT NULL,
    section_id TEXT NOT NULL,
    passage_id TEXT NOT NULL,
    start_offset INTEGER NOT NULL,
    end_offset INTEGER NOT NULL,
    quote TEXT NOT NULL,
    text TEXT NOT NULL,
    asked_at TEXT NOT NULL,
    status TEXT NOT NULL,
    FOREIGN KEY (opinion_id) REFERENCES opinions(opinion_id) ON DELETE CASCADE,
    FOREIGN KEY (passage_id) REFERENCES passages(passage_id) ON DELETE CASCADE
);
