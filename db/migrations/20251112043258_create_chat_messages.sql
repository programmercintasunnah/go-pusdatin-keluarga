-- migrate:up
CREATE TABLE IF NOT EXISTS chat_messages (
    id SERIAL PRIMARY KEY,
    username VARCHAR(100),
    message TEXT,
    sent_at TIMESTAMP DEFAULT NOW()
);

-- migrate:down
DROP TABLE IF EXISTS chat_messages;
