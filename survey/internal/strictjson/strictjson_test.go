package strictjson

import (
	"strings"
	"testing"
)

func TestDecode(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		wantErr string
	}{
		{name: "valid", data: `{"name":"value"}`},
		{name: "duplicate", data: `{"name":"first","name":"second"}`, wantErr: "duplicate key"},
		{name: "unknown", data: `{"name":"value","extra":true}`, wantErr: "unknown field"},
		{name: "multiple", data: `{"name":"value"} {"name":"other"}`, wantErr: "multiple JSON values"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var destination struct {
				Name string `json:"name"`
			}
			err := Decode([]byte(test.data), &destination)
			if test.wantErr == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("Decode() error = %v, want substring %q", err, test.wantErr)
			}
		})
	}
}
