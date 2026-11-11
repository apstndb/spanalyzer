package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"cloud.google.com/go/spanner"
)

// The broad matrix discovered that OR IGNORE is accepted. These controls
// separate attempted rows from actually inserted rows, including source queries.
func (q *queueMatrixRunner) ignoreCases() error {
	f := queueFixtures()[0]
	ignoreSQL := func(sql string) string { return strings.Replace(sql, "INSERT INTO", "INSERT OR IGNORE INTO", 1) }
	for _, state := range []string{"absent", "same", "changed"} {
		for _, delivery := range []bool{false, true} {
			k := q.key(f)
			if state != "absent" {
				if err := q.seedMessages(f, []spanner.Key{k}); err != nil {
					return err
				}
			}
			sql := q.insertSQL(f, []spanner.Key{k}, delivery)
			if state == "changed" {
				sql = strings.Replace(sql, "b'x'", "b'changed'", 1)
			}
			q.run(fmt.Sprintf("ignore/one/state=%s/delivery=%v", state, delivery), f.name, "dml_insert_ignore", state, "observe", nil, ignoreSQL(sql))
		}
	}
	for present := range 4 {
		for _, delivery := range []bool{false, true} {
			keys := []spanner.Key{q.key(f), q.key(f), q.key(f)}
			if err := q.seedMessages(f, keys[:present]); err != nil {
				return err
			}
			q.run(fmt.Sprintf("ignore/mixed3/present=%d/delivery=%v", present, delivery), f.name, "dml_insert_ignore", fmt.Sprintf("%d of 3 target rows present", present), "observe", nil, ignoreSQL(q.insertSQL(f, keys, delivery)))
		}
	}
	for _, present := range []bool{false, true} {
		for _, different := range []bool{false, true} {
			k := q.key(f)
			if present {
				if err := q.seedMessages(f, []spanner.Key{k}); err != nil {
					return err
				}
			}
			sql := ignoreSQL(q.insertSQL(f, []spanner.Key{k, k}, true))
			if different {
				at := strings.LastIndex(sql, "b'x'")
				sql = sql[:at] + "b'y'" + sql[at+4:]
			}
			q.run(fmt.Sprintf("ignore/repeated-key/present=%v/different=%v", present, different), f.name, "dml_insert_ignore", fmt.Sprintf("target present=%v; two values rows", present), "observe", nil, sql)
		}
	}
	for _, present := range []bool{false, true} {
		k := q.key(f)
		if present {
			if err := q.seedMessages(f, []spanner.Key{k}); err != nil {
				return err
			}
		}
		sql := ignoreSQL(q.insertSQL(f, []spanner.Key{k}, false))
		sql = strings.Replace(sql, "b'x'", "NULL", 1)
		q.run(fmt.Sprintf("ignore/null-payload/present=%v", present), f.name, "dml_insert_ignore", fmt.Sprintf("target present=%v", present), "observe", nil, sql)
	}
	for _, mode := range []string{"insert", "ignore"} {
		for _, tc := range []struct{ n, present int }{{0, 0}, {1, 0}, {3, 0}, {3, 1}, {3, 3}} {
			if mode == "insert" && tc.present > 0 {
				continue
			}
			var keys []spanner.Key
			var literals []string
			for range tc.n {
				k := q.key(f)
				keys = append(keys, k)
				literals = append(literals, matrixKeySQL(k))
			}
			if err := q.seedMessages(f, keys[:tc.present]); err != nil {
				return err
			}
			array := "ARRAY<STRING>[]"
			if tc.n > 0 {
				array = "[" + strings.Join(literals, ",") + "]"
			}
			sql := "INSERT INTO Messages(Id,Payload) SELECT Id,b'x' FROM UNNEST(" + array + ") AS Id"
			if mode == "ignore" {
				sql = ignoreSQL(sql)
			}
			q.run(fmt.Sprintf("select/%s/n=%d/present=%d", mode, tc.n, tc.present), f.name, "dml_insert_select", fmt.Sprintf("%d source rows; %d targets present", tc.n, tc.present), "observe", nil, sql)
		}
	}
	for _, present := range []bool{false, true} {
		k := q.key(f)
		if present {
			if err := q.seedMessages(f, []spanner.Key{k}); err != nil {
				return err
			}
		}
		sql := ignoreSQL(q.insertSQL(f, []spanner.Key{k}, true))
		q.run(fmt.Sprintf("ignore/then-delete/present=%v", present), f.name, "dml_mixed", fmt.Sprintf("target present=%v", present), "observe", nil, sql, "DELETE FROM Messages WHERE "+queueKeyPredicate(f, k))
	}
	k := q.key(f)
	sql := ignoreSQL(q.insertSQL(f, []spanner.Key{k}, true))
	q.run("ignore/two-statements", f.name, "dml_insert_ignore", "absent before first statement", "observe", nil, sql, sql)
	for _, present := range []bool{false, true} {
		for _, assert := range []int{0, 1} {
			k := q.key(f)
			if present {
				if err := q.seedMessages(f, []spanner.Key{k}); err != nil {
					return err
				}
			}
			q.run(fmt.Sprintf("delete/assert-rows/present=%v/assert=%d", present, assert), f.name, "dml_delete", fmt.Sprintf("target present=%v", present), "observe", nil, "DELETE FROM Messages WHERE "+queueKeyPredicate(f, k)+fmt.Sprintf(" ASSERT_ROWS_MODIFIED %d", assert))
		}
	}
	for _, present := range []bool{false, true} {
		for _, action := range []string{"ON CONFLICT DO NOTHING", "ON CONFLICT (Id) DO NOTHING", "ON CONFLICT (Id) DO UPDATE SET Payload=excluded.Payload"} {
			k := q.key(f)
			if present {
				if err := q.seedMessages(f, []spanner.Key{k}); err != nil {
					return err
				}
			}
			q.run(fmt.Sprintf("conflict/present=%v/%s", present, action), f.name, "dml_conflict", fmt.Sprintf("target present=%v", present), "observe", nil, q.insertSQL(f, []spanner.Key{k}, true)+" "+action)
		}
	}
	return q.renewReadWrite()
}

func printQueueIgnorePlan() error {
	q := newQueueMatrixRunner(context.Background(), nil, &result{}, nil, nil, nil)
	if err := q.ignoreCases(); err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(struct {
		Status string                `json:"status"`
		Cases  []queueMatrixPlanCase `json:"cases"`
	}{"prepared_not_executed", q.plan})
}
