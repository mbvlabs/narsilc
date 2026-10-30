CREATE TABLE events (
  id          BIGSERIAL PRIMARY KEY,
  release_id  BIGINT,
  occurred_at timestamptz NOT NULL,
  archived_at timestamptz,
  note        text NOT NULL
);
