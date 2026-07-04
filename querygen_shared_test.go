package spanalyzer

import (
	"testing"

	"cloud.google.com/go/spanner/apiv1/spannerpb"
)

func TestTypeSpecSQLInterval(t *testing.T) {
	got, err := typeSpecSQL(&TypeSpec{Code: spannerpb.TypeCode_INTERVAL})
	if err != nil {
		t.Fatalf("typeSpecSQL(INTERVAL) error = %v", err)
	}
	if got != "INTERVAL" {
		t.Fatalf("typeSpecSQL(INTERVAL) = %q, want INTERVAL", got)
	}
}
