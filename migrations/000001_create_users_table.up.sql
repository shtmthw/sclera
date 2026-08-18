CREATE SCHEMA IF NOT EXISTS sclera;

CREATE TABLE sclera.users (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    age INT NOT NULL,
    favouriteTopics TEXT[] NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);