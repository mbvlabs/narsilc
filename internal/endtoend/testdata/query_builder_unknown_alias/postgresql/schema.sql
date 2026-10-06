CREATE TABLE users (
  id    BIGSERIAL PRIMARY KEY,
  email text NOT NULL
);

CREATE TABLE tokens (
  id      UUID PRIMARY KEY,
  user_id bigint NOT NULL REFERENCES users (id),
  scope   text NOT NULL
);
