CREATE TABLE tag (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) UNIQUE NOT NULL
);

CREATE TABLE note (
    id SERIAL PRIMARY KEY,
    text TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    modified_at TIMESTAMP NOT NULL DEFAULT NOW(),
    date TIMESTAMP NOT NULL DEFAULT NOW(),
    parent_id INTEGER REFERENCES note(id),
    reply_count INTEGER DEFAULT 0,
    is_deleted BOOLEAN DEFAULT FALSE
);

CREATE TABLE note_to_tag (
    id SERIAL PRIMARY KEY,
    note_id INTEGER NOT NULL REFERENCES note(id),
    tag_id INTEGER NOT NULL REFERENCES tag(id),
    UNIQUE (note_id, tag_id)
);