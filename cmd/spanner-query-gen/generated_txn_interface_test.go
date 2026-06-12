package main

import (
	"context"

	"cloud.google.com/go/spanner"
)

// The SpannerQueryTransaction interface emitted by the querygen generator
// must stay satisfied by all three Cloud Spanner transaction types. The
// assertions live in this module rather than internal/querygen so the root
// analyzer module does not take a dependency on the spanner client package,
// which this module already requires.
type generatedSpannerQueryTransaction interface {
	Query(ctx context.Context, statement spanner.Statement) *spanner.RowIterator
}

var (
	_ generatedSpannerQueryTransaction = (*spanner.ReadOnlyTransaction)(nil)
	_ generatedSpannerQueryTransaction = (*spanner.ReadWriteTransaction)(nil)
	_ generatedSpannerQueryTransaction = (*spanner.BatchReadOnlyTransaction)(nil)
)
