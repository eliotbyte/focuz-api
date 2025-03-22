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
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
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
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

ALTER TABLE note
    ADD COLUMN topic_id INTEGER REFERENCES topic(id);
