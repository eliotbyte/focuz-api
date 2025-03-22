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

INSERT INTO role (name) VALUES ('owner'), ('guest');

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

INSERT INTO topic_type (name) VALUES ('diary'), ('dashboard');

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

CREATE TABLE note_to_tag (
    id SERIAL PRIMARY KEY,
    note_id INTEGER NOT NULL REFERENCES note(id),
    tag_id INTEGER NOT NULL REFERENCES tag(id),
    UNIQUE (note_id, tag_id)
);
