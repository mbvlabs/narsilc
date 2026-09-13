CREATE TABLE authors (
  id   BIGSERIAL PRIMARY KEY,
  name text NOT NULL,
  bio  text
);

CREATE TABLE posts (
  id        BIGSERIAL PRIMARY KEY,
  author_id BIGINT NOT NULL REFERENCES authors (id),
  title     text NOT NULL
);
