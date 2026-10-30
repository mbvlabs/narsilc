CREATE TABLE left_items (
  id   BIGSERIAL PRIMARY KEY,
  name text NOT NULL
);

CREATE TABLE right_items (
  id      BIGSERIAL PRIMARY KEY,
  left_id BIGINT NOT NULL
);
