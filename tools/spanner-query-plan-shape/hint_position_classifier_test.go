package main

import (
	"strings"
	"testing"

	"cloud.google.com/go/spanner"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var hintPositionRejectionMarkers = []string{
	"hint cannot be used at beginning of path pattern",
	"hint cannot be used in between two graphedgepatterns",
	"hints cannot be specified on in clause with value list",
	"hints cannot be specified on in clause with unnest",
	"hints cannot be specified on like clause with value list",
	"hints cannot be specified on any/some/all clause with value list",
	"hints on set operations must appear on the first operation",
}

func classifyHintPositionResult(err error) (hintPositionExpectation, string) {
	if err == nil {
		return hintPositionAccepted, "executed"
	}
	description := spanner.ErrDesc(err)
	lower := strings.ToLower(description)
	if strings.Contains(lower, "syntax error") || isKnownHintPositionRejection(lower) {
		return hintPositionRejected, description
	}
	switch spanner.ErrCode(err) {
	case codes.InvalidArgument:
		if strings.Contains(lower, "unsupported") ||
			strings.Contains(lower, "not supported") ||
			strings.Contains(lower, "not found") {
			return hintPositionAccepted, description
		}
		return "inconclusive", description
	case codes.FailedPrecondition, codes.NotFound, codes.Unimplemented:
		return hintPositionAccepted, description
	default:
		return "inconclusive", description
	}
}

func isKnownHintPositionRejection(description string) bool {
	// Keep this list tied to diagnostics observed for the rejected audit cases.
	// A broader phrase match could turn a future hint-validation error into a
	// false rejection instead of the safer inconclusive result.
	for _, marker := range hintPositionRejectionMarkers {
		if strings.Contains(description, marker) {
			return true
		}
	}
	return false
}

func TestClassifyHintPositionResult(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want hintPositionExpectation
	}{
		{name: "executed", want: hintPositionAccepted},
		{name: "syntax error", err: status.Error(codes.InvalidArgument, "Syntax error: bad placement"), want: hintPositionRejected},
		{name: "GQL subpath leading", err: status.Error(codes.InvalidArgument, "Hint cannot be used at beginning of path pattern"), want: hintPositionRejected},
		{name: "GQL between edges", err: status.Error(codes.InvalidArgument, "Hint cannot be used in between two GraphEdgePatterns"), want: hintPositionRejected},
		{name: "IN value list", err: status.Error(codes.InvalidArgument, "HINTs cannot be specified on IN clause with value list"), want: hintPositionRejected},
		{name: "IN UNNEST", err: status.Error(codes.InvalidArgument, "HINTs cannot be specified on IN clause with UNNEST"), want: hintPositionRejected},
		{name: "LIKE value list", err: status.Error(codes.InvalidArgument, "HINTs cannot be specified on LIKE clause with value list"), want: hintPositionRejected},
		{name: "quantified value list", err: status.Error(codes.InvalidArgument, "HINTs cannot be specified on ANY/SOME/ALL clause with value list"), want: hintPositionRejected},
		{name: "set operation restriction", err: status.Error(codes.InvalidArgument, "Hints on set operations must appear on the first operation"), want: hintPositionRejected},
		{name: "unsupported hint", err: status.Error(codes.InvalidArgument, "Unsupported hint: a"), want: hintPositionAccepted},
		{name: "feature not supported", err: status.Error(codes.InvalidArgument, "LIKE ALL is not supported"), want: hintPositionAccepted},
		{name: "name resolution", err: status.Error(codes.NotFound, "Table-valued function not found"), want: hintPositionAccepted},
		{name: "unimplemented", err: status.Error(codes.Unimplemented, "feature unavailable"), want: hintPositionAccepted},
		{name: "unknown invalid argument", err: status.Error(codes.InvalidArgument, "new unclassified validation failure"), want: "inconclusive"},
		{name: "unrecognized hint cannot be used", err: status.Error(codes.InvalidArgument, "Hint cannot be used with this feature"), want: "inconclusive"},
		{name: "unrecognized hints must", err: status.Error(codes.InvalidArgument, "Hints on join methods must use a supported value"), want: "inconclusive"},
		{name: "transport", err: status.Error(codes.Unavailable, "connection lost"), want: "inconclusive"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := classifyHintPositionResult(tt.err)
			if got != tt.want {
				t.Fatalf("classifyHintPositionResult() = %q, want %q", got, tt.want)
			}
		})
	}
}
