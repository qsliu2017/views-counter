CREATE TABLE count (
  id SERIAL PRIMARY KEY,
  payload TEXT NOT NULL,
  ts BIGINT NOT NULL DEFAULT extract(epoch from now())
);
