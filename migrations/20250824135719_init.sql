-- +goose Up
-- +goose StatementBegin

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TYPE message_status_enum AS ENUM ('SENT', 'DELIVERED', 'READ');

CREATE TABLE users (
    id CHAR(26) PRIMARY KEY,
    username VARCHAR(100) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER set_users_updated_at
BEFORE UPDATE ON users
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE chats (
    id CHAR(26) PRIMARY KEY,
    name VARCHAR(50) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER set_chats_updated_at
BEFORE UPDATE ON chats
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE messages (
    id CHAR(26) PRIMARY KEY,
    idempotency_key VARCHAR(100) NOT NULL UNIQUE,
    user_id CHAR(26) NOT NULL,
    chat_id CHAR(26) NOT NULL,
    content TEXT NOT NULL,
    status message_status_enum NOT NULL DEFAULT 'SENT',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (chat_id) REFERENCES chats(id) ON DELETE CASCADE
);

CREATE INDEX idx_messages_user_id ON messages (user_id);
CREATE INDEX idx_messages_chat_id ON messages (chat_id);
CREATE INDEX idx_messages_created_at ON messages (created_at);
CREATE INDEX idx_messages_idempotency_key ON messages (idempotency_key);
CREATE INDEX idx_messages_chat_created_at ON messages (chat_id, created_at);

CREATE TRIGGER set_messages_updated_at
BEFORE UPDATE ON messages
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE users_chats (
    id CHAR(26) PRIMARY KEY,
    user_id CHAR(26) NOT NULL,
    chat_id CHAR(26) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (chat_id) REFERENCES chats(id) ON DELETE CASCADE,
    
    UNIQUE (user_id, chat_id)
);

CREATE INDEX idx_users_chats_user_id ON users_chats (user_id);
CREATE INDEX idx_users_chats_chat_id ON users_chats (chat_id);

CREATE TRIGGER set_users_chats_updated_at
BEFORE UPDATE ON users_chats
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TRIGGER IF EXISTS set_users_updated_at ON users;
DROP TRIGGER IF EXISTS set_chats_updated_at ON chats;
DROP TRIGGER IF EXISTS set_messages_updated_at ON messages;
DROP TRIGGER IF EXISTS set_users_chats_updated_at ON users_chats;

DROP FUNCTION IF EXISTS set_updated_at;

DROP TABLE IF EXISTS users_chats;
DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS chats;
DROP TABLE IF EXISTS users;

DROP TYPE IF EXISTS message_status_enum;

-- +goose StatementEnd
