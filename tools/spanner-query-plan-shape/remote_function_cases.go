package main

const remoteFunctionDDL = `
CREATE SCHEMA spanalyzer_remote;

CREATE TABLE RemoteFunctionInputs (
  InputId INT64 NOT NULL,
  X INT64,
  Y INT64,
) PRIMARY KEY (InputId);

CREATE FUNCTION spanalyzer_remote.remote_add(x INT64, y INT64) RETURNS INT64
NOT DETERMINISTIC LANGUAGE REMOTE
OPTIONS (
  endpoint = "https://spanalyzer-remote-test-uc.a.run.app",
  max_batching_rows = 10
);
`

var remoteFunctionQueries = []queryCase{
	{
		Label: "remote-function/literal-input",
		SQL:   `SELECT spanalyzer_remote.remote_add(1, 2) AS total`,
	},
	{
		Label: "remote-function/table-input",
		SQL:   `SELECT InputId, spanalyzer_remote.remote_add(X, Y) AS total FROM RemoteFunctionInputs`,
	},
}
