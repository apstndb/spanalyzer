package astconv

import (
	"strings"
	"testing"

	"cloud.google.com/go/spanner"
	"github.com/apstndb/spanner-emulator-survey/infoschem"
)

func TestToDDLStatements_RemainingNamedSchemaObjectsFailExplicitly(t *testing.T) {
	tests := []struct {
		name   string
		schema *Schema
		want   string
	}{
		{
			name: "change stream",
			schema: &Schema{ChangeStreams: []*infoschem.ChangeStream{
				{ChangeStreamSchema: "app", ChangeStreamName: "Changes", All: true},
			}},
			want: "unsupported named-schema change stream",
		},
		{
			name: "change stream watched table",
			schema: &Schema{ChangeStreamTables: []*infoschem.ChangeStreamTable{
				{ChangeStreamName: "Changes", TableSchema: "app", TableName: "Records", AllColumns: true},
			}},
			want: "unsupported named-schema watched table",
		},
		{
			name: "interleaved index",
			schema: &Schema{Indexes: []*infoschem.Index{
				{
					TableSchema:     "app",
					TableName:       "Records",
					IndexName:       "ByValue",
					IndexType:       "INDEX",
					ParentTableName: "Parents",
				},
			}},
			want: "unsupported named-schema interleaved index",
		},
		{
			name: "interleaved search index",
			schema: &Schema{Indexes: []*infoschem.Index{
				{
					TableSchema:     "app",
					TableName:       "Records",
					IndexName:       "SearchRecords",
					IndexType:       "SEARCH",
					ParentTableName: "Parents",
				},
			}},
			want: "unsupported named-schema interleaved index",
		},
		{
			name: "model",
			schema: &Schema{Models: []*infoschem.Model{
				{ModelSchema: "app", ModelName: "Classifier"},
			}},
			want: "unsupported named-schema model",
		},
		{
			name: "property graph row",
			schema: &Schema{PropertyGraphs: []*infoschem.PropertyGraph{
				{PropertyGraphSchema: "app", PropertyGraphName: "Graph"},
			}},
			want: "unsupported named-schema property graph",
		},
		{
			name: "property graph metadata",
			schema: &Schema{PropertyGraphs: []*infoschem.PropertyGraph{
				{
					PropertyGraphName: "Graph",
					PropertyGraphMetadataJSON: spanner.NullJSON{Valid: true, Value: &propertyGraphMetadata{
						Schema: "app",
						Name:   "Graph",
					}},
				},
			}},
			want: "unsupported named-schema property graph",
		},
		{
			name: "property graph element table",
			schema: &Schema{PropertyGraphs: []*infoschem.PropertyGraph{
				{
					PropertyGraphName: "Graph",
					PropertyGraphMetadataJSON: spanner.NullJSON{Valid: true, Value: &propertyGraphMetadata{
						Name: "Graph",
						NodeTables: []*propertyGraphElementTable{{
							Name:           "Node",
							BaseSchemaName: "app",
							BaseTableName:  "Records",
						}},
					}},
				},
			}},
			want: "unsupported named-schema property graph element table",
		},
		{
			name: "statistics package",
			schema: &Schema{SpannerStatistics: []*infoschem.SpannerStatistic{
				{SchemaName: "app", PackageName: "Stats"},
			}},
			want: "unsupported named-schema statistics package",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.schema.ToDDLStatements()
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("ToDDLStatements error = %v, want %q", err, tt.want)
			}
		})
	}
}
