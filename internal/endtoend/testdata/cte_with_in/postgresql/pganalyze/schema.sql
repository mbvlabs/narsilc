-- https://github.com/mbvlabs/narsilc/issues/2153

CREATE TABLE L (
  id   BIGSERIAL PRIMARY KEY,
  parent_id int null
);

CREATE TABLE T (
  id   BIGSERIAL PRIMARY KEY,
  L_ID int,
  F varchar(256)
);
