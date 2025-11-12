CREATE TABLE IF NOT EXISTS weather (
    id SERIAL PRIMARY KEY,
    city VARCHAR(100),
    temperature FLOAT,
    weather_desc TEXT,
    collected_at TIMESTAMP
);

CREATE TABLE IF NOT EXISTS chat_messages (
    id SERIAL PRIMARY KEY,
    username VARCHAR(100),
    message TEXT,
    sent_at TIMESTAMP DEFAULT NOW()
);