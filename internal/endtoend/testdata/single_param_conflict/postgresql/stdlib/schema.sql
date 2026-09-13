-- Example queries for sqlc
CREATE TABLE authors (
  id   BIGSERIAL PRIMARY KEY,
  name TEXT      NOT NULL,
  bio  text
);

-- https://github.com/mbvlabs/narsilc/issues/1290
CREATE TABLE users (
  sub UUID PRIMARY KEY
);
