CREATE TABLE Events (
  Shard INT64 NOT NULL,
  Id INT64 NOT NULL,
  Active STRING(MAX),
  Note STRING(MAX),
  Extra STRING(MAX)
) PRIMARY KEY (Shard, Id);
CREATE INDEX ByActive ON Events (Active) STORING (Note);
