package spanalyzer

import "testing"

func TestMutationCountFKPrimaryKeyCoverageBounds(t *testing.T) {
	for _, tc := range []struct {
		name, ddl, sql string
		lo, hi         int64
	}{
		{
			name: "nullable referencing PK",
			ddl:  `CREATE TABLE P(Id INT64 NOT NULL) PRIMARY KEY(Id); CREATE TABLE C(Id INT64,V STRING(MAX),CONSTRAINT FK FOREIGN KEY(Id) REFERENCES P(Id)) PRIMARY KEY(Id);`,
			sql:  `INSERT INTO C(Id,V) VALUES(1,'x')`, lo: 2, hi: 3,
		},
		{
			name: "required referencing PK",
			ddl:  `CREATE TABLE P(Id INT64 NOT NULL) PRIMARY KEY(Id); CREATE TABLE C(Id INT64 NOT NULL,V STRING(MAX),CONSTRAINT FK FOREIGN KEY(Id) REFERENCES P(Id)) PRIMARY KEY(Id);`,
			sql:  `INSERT INTO C(Id,V) VALUES(1,'x')`, lo: 2, hi: 2,
		},
		{
			name: "referenced key reorders referencing PK",
			ddl:  `CREATE TABLE P(A INT64 NOT NULL,B INT64 NOT NULL) PRIMARY KEY(A,B); CREATE TABLE C(X INT64 NOT NULL,Y INT64 NOT NULL,V STRING(MAX),CONSTRAINT FK FOREIGN KEY(X,Y) REFERENCES P(B,A)) PRIMARY KEY(X,Y);`,
			sql:  `INSERT INTO C(X,Y,V) VALUES(2,1,'x')`, lo: 3, hi: 4,
		},
		{
			name: "unchanged composite PK prefix",
			ddl:  `CREATE TABLE P(A INT64 NOT NULL,B INT64 NOT NULL) PRIMARY KEY(A,B); CREATE TABLE C(X INT64 NOT NULL,Y INT64 NOT NULL,V STRING(MAX),CONSTRAINT FK FOREIGN KEY(X,Y) REFERENCES P(A,B)) PRIMARY KEY(X,Y);`,
			sql:  `INSERT INTO C(X,Y,V) VALUES(1,2,'x')`, lo: 3, hi: 3,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := mutationTestAnalyzer(t, tc.ddl)
			r, err := a.AnalyzeDML([]string{tc.sql}, MutationCountOptions{})
			if err != nil {
				t.Fatal(err)
			}
			assertMutationInterval(t, r.Bounds.Count, tc.lo, mutationTestInt(tc.hi))
		})
	}
}

func TestMutationCountNullableFKPKObservedIndexRefinesBounds(t *testing.T) {
	a := mutationTestAnalyzer(t, `CREATE TABLE P(Id INT64 NOT NULL) PRIMARY KEY(Id); CREATE TABLE C(Id INT64,V STRING(MAX),CONSTRAINT FK FOREIGN KEY(Id) REFERENCES P(Id)) PRIMARY KEY(Id);`)
	payload := []byte(`[{"insert":{"table":"C","columns":["Id","V"],"values":[["1","x"]]}}]`)
	r, err := a.AnalyzeMutationsJSON(payload, MutationCountOptions{})
	if err != nil {
		t.Fatal(err)
	}
	assertMutationInterval(t, r.Bounds.Count, 2, mutationTestInt(3))
	a, err = a.WithObservedIndexes(a.SchemaSHA256(), []MutationCountIndexObservation{{Table: "C", Name: "observed_backing_index", Type: "INDEX", State: "READ_WRITE", Columns: []string{"Id"}, Managed: true, NullFiltered: true}})
	if err != nil {
		t.Fatal(err)
	}
	r, err = a.AnalyzeMutationsJSON(payload, MutationCountOptions{})
	if err != nil {
		t.Fatal(err)
	}
	// The pinned Omni controls observed 3, versus 2 with NOT NULL. These local
	// regressions check the model and supplied metadata; they perform no RPCs.
	assertMutationInterval(t, r.Bounds.Count, 3, mutationTestInt(3))
}
