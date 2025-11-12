-- migrate:up
CREATE TABLE weather (
    id SERIAL PRIMARY KEY,
    city TEXT NOT NULL,
    temperature DOUBLE PRECISION,
    weather_desc TEXT,
    collected_at TIMESTAMP DEFAULT NOW()
);

-- migrate:down
DROP TABLE IF EXISTS weather;
