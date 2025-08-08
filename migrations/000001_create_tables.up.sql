CREATE EXTENSION IF NOT EXISTS pgroonga;

CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL CHECK (LENGTH(username) >= 3 AND LENGTH(username) <= 50),
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE role (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE
);

CREATE TABLE space (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    owner_id INTEGER REFERENCES users(id),
    is_deleted BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    modified_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE user_to_space (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id),
    space_id INTEGER NOT NULL REFERENCES space(id),
    role_id INTEGER NOT NULL REFERENCES role(id),
    UNIQUE (user_id, space_id)
);

CREATE TABLE topic_type (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE
);

CREATE TABLE topic (
    id SERIAL PRIMARY KEY,
    space_id INTEGER NOT NULL REFERENCES space(id),
    name VARCHAR(255) NOT NULL,
    type_id INTEGER NOT NULL REFERENCES topic_type(id),
    is_deleted BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    modified_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE tag (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) UNIQUE NOT NULL
);

CREATE TABLE tag_to_space_topic (
    id SERIAL PRIMARY KEY,
    tag_id INTEGER NOT NULL REFERENCES tag(id),
    space_id INTEGER NOT NULL REFERENCES space(id),
    topic_id INTEGER REFERENCES topic(id),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE (tag_id, space_id, topic_id)
);

CREATE TABLE note (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id),
    text TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    modified_at TIMESTAMP NOT NULL DEFAULT NOW(),
    date TIMESTAMP NOT NULL DEFAULT NOW(),
    parent_id INTEGER REFERENCES note(id),
    reply_count INTEGER DEFAULT 0,
    is_deleted BOOLEAN DEFAULT FALSE,
    topic_id INTEGER REFERENCES topic(id)
);

CREATE INDEX pgroonga_note_text_index ON note USING pgroonga (text);

CREATE TABLE note_to_tag (
    id SERIAL PRIMARY KEY,
    note_id INTEGER NOT NULL REFERENCES note(id),
    tag_id INTEGER NOT NULL REFERENCES tag(id),
    UNIQUE (note_id, tag_id)
);

CREATE TABLE activity_type_category (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE
);

CREATE TABLE activity_types (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    value_type VARCHAR(50) NOT NULL,
    min_value DOUBLE PRECISION,
    max_value DOUBLE PRECISION,
    aggregation VARCHAR(50) NOT NULL,
    space_id INTEGER REFERENCES space(id),
    is_default BOOLEAN NOT NULL DEFAULT false,
    is_deleted BOOLEAN NOT NULL DEFAULT false,
    unit VARCHAR(50),
    category_id INTEGER REFERENCES activity_type_category(id),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    modified_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE (name, space_id)
);

CREATE TABLE activities (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id),
    type_id INTEGER NOT NULL REFERENCES activity_types(id),
    value JSONB NOT NULL,
    note_id INTEGER REFERENCES note(id),
    is_deleted BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    modified_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE attachments (
    id VARCHAR(36) PRIMARY KEY,
    note_id INTEGER NOT NULL REFERENCES note(id),
    file_name VARCHAR(255) NOT NULL,
    file_type VARCHAR(100) NOT NULL,
    file_size BIGINT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    modified_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE chart (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id),
    topic_id INTEGER NOT NULL REFERENCES topic(id),
    kind VARCHAR(50) NOT NULL,
    activity_type_id INTEGER NOT NULL REFERENCES activity_types(id),
    period VARCHAR(50) NOT NULL,
    is_deleted BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    modified_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Insert dashboard topic type
INSERT INTO topic_type (name) VALUES ('dashboard') ON CONFLICT (name) DO NOTHING;

-- Performance indexes
CREATE INDEX IF NOT EXISTS idx_note_topic_id ON note(topic_id);
CREATE INDEX IF NOT EXISTS idx_note_date ON note(date);
CREATE INDEX IF NOT EXISTS idx_note_created_at ON note(created_at);
CREATE INDEX IF NOT EXISTS idx_topic_space_id ON topic(space_id);
CREATE INDEX IF NOT EXISTS idx_user_to_space_user_space ON user_to_space(user_id, space_id);
CREATE INDEX IF NOT EXISTS idx_attachments_note_id ON attachments(note_id);
CREATE INDEX IF NOT EXISTS idx_activities_note_id ON activities(note_id);
CREATE INDEX IF NOT EXISTS idx_activities_type_id ON activities(type_id);
CREATE INDEX IF NOT EXISTS idx_activity_types_space_id ON activity_types(space_id);
