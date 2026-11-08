package infoschem

import (
	"errors"
	"slices"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestColumnsNamedByRejection(t *testing.T) {
	candidates := map[string]bool{"IS_INHERITED": true, "OPTION_NAME": true}
	tests := []struct {
		name string
		err  error
		want []string
	}{
		{
			name: "unrecognized name form",
			err:  status.Error(codes.InvalidArgument, "Unrecognized name: IS_INHERITED [at 1:116]\n...`OPTION_VALUE`, `IS_INHERITED` FROM INFORMATI..."),
			want: []string{"IS_INHERITED"},
		},
		{
			name: "not found inside alias form",
			err:  status.Error(codes.InvalidArgument, "Name IS_INHERITED not found inside t [at 1:561]"),
			want: []string{"IS_INHERITED"},
		},
		{
			name: "named column is not a candidate",
			err:  status.Error(codes.InvalidArgument, "Unrecognized name: SOMETHING_ELSE"),
			want: nil,
		},
		{
			name: "non InvalidArgument code",
			err:  status.Error(codes.Unavailable, "Unrecognized name: IS_INHERITED"),
			want: nil,
		},
		{
			name: "closed transaction is not a rejection",
			err:  status.Error(codes.InvalidArgument, closedTransactionErr),
			want: nil,
		},
		{
			name: "plain error",
			err:  errors.New("Unrecognized name: IS_INHERITED"),
			want: nil,
		},
		{
			name: "nil error",
			err:  nil,
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ColumnsNamedByRejection(tt.err, candidates); !slices.Equal(got, tt.want) {
				t.Fatalf("ColumnsNamedByRejection() = %v, want %v", got, tt.want)
			}
		})
	}
}
