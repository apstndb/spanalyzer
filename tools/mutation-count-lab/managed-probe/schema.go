package main

import (
	"context"

	"cloud.google.com/go/spanner"
)

type schemaPhase struct {
	DDL          []string      `json:"ddl"`
	Indexes      []metadata    `json:"indexes"`
	QueueColumns []queueColumn `json:"queue_columns,omitempty"`
}
type queueColumn struct {
	Table, Name, Type, Nullable, Generated string
	Default                                *string
}

func captureSchemaPhase(ctx context.Context, c *managedClients, r *result) error {
	p := schemaPhase{DDL: append([]string(nil), r.DDL...)}
	for _, extra := range r.Extra {
		if extra.Status == "OK" {
			p.DDL = append(p.DDL, extra.DDL)
		}
	}
	iter := c.Client.Single().Query(ctx, spanner.NewStatement("SELECT TABLE_NAME, INDEX_NAME, INDEX_TYPE, INDEX_STATE, IS_UNIQUE, IS_NULL_FILTERED, SPANNER_IS_MANAGED FROM INFORMATION_SCHEMA.INDEXES WHERE TABLE_SCHEMA = '' AND INDEX_TYPE <> 'PRIMARY_KEY' ORDER BY TABLE_NAME,INDEX_NAME"))
	err := iter.Do(func(row *spanner.Row) error {
		var m metadata
		if err := row.Columns(&m.Table, &m.Name, &m.Type, &m.State, &m.Unique, &m.NullFiltered, &m.Managed); err != nil {
			return err
		}
		p.Indexes = append(p.Indexes, m)
		return nil
	})
	if err != nil {
		return err
	}
	for n := range p.Indexes {
		m := &p.Indexes[n]
		q := spanner.NewStatement("SELECT COLUMN_NAME,ORDINAL_POSITION FROM INFORMATION_SCHEMA.INDEX_COLUMNS WHERE TABLE_SCHEMA='' AND TABLE_NAME=@t AND INDEX_NAME=@i ORDER BY ORDINAL_POSITION")
		q.Params = map[string]interface{}{"t": m.Table, "i": m.Name}
		err = c.Client.Single().Query(ctx, q).Do(func(row *spanner.Row) error {
			var name string
			var ord spanner.NullInt64
			if err := row.Columns(&name, &ord); err != nil {
				return err
			}
			if ord.Valid {
				m.Columns = append(m.Columns, name)
			} else {
				m.Stored = append(m.Stored, name)
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	err = c.Client.Single().Query(ctx, spanner.NewStatement("SELECT TABLE_NAME,COLUMN_NAME,SPANNER_TYPE,IS_NULLABLE,IS_GENERATED,COLUMN_DEFAULT FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA='' AND TABLE_NAME IN ('Messages','TwoKeyMessages','ThreeKeyMessages','TextMessages','JsonMessages','ProtoMessages','ReceiveMessages','ChildMessages','DeepMessages') ORDER BY TABLE_NAME,ORDINAL_POSITION")).Do(func(row *spanner.Row) error {
		var col queueColumn
		var d spanner.NullString
		if err := row.Columns(&col.Table, &col.Name, &col.Type, &col.Nullable, &col.Generated, &d); err != nil {
			return err
		}
		if d.Valid {
			v := d.StringVal
			col.Default = &v
		}
		p.QueueColumns = append(p.QueueColumns, col)
		return nil
	})
	if err != nil {
		return err
	}
	r.SchemaPhases = append(r.SchemaPhases, p)
	return nil
}
