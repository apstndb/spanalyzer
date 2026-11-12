package plancontract

func celAccessPaths(paths []AccessPath) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(paths))
	for _, path := range paths {
		var filter interface{}
		if path.FilterIndex != nil {
			filter = *path.FilterIndex
		}
		var rangeSize, keyCount, nullFiltered interface{}
		if path.RangeSeekableKeySize != nil {
			rangeSize = *path.RangeSeekableKeySize
		}
		if path.DeclaredKeyCount != nil {
			keyCount = *path.DeclaredKeyCount
		}
		if path.NullFiltered != nil {
			nullFiltered = *path.NullFiltered
		}
		selectors := path.InferredConstraintSelectors
		if selectors == nil {
			selectors = []string{}
		}
		out = append(out, map[string]interface{}{
			"catalog_index_name": path.CatalogIndexName, "index_origin": path.IndexOrigin,
			"inferred_constraint_selectors": selectors,
			"managed":                       accessPathNullableBool(path.Managed),
			"scan_index":                    path.ScanIndex, "filter_index": filter,
			"scan_target": path.ScanTarget, "scan_type": path.ScanType,
			"scan_target_kind": path.ScanTargetKind, "base_table": path.BaseTable,
			"full_scan": path.FullScan, "conditions_complete": path.ConditionsComplete,
			"has_seek_condition": path.HasSeekCondition, "has_residual_condition": path.HasResidualCondition,
			"has_timestamp_condition": path.HasTimestampCondition, "has_search_condition": path.HasSearchCondition,
			"seek_conditions": celScanConditions(path.SeekConditions), "residual_conditions": celScanConditions(path.ResidualConditions),
			"timestamp_conditions": celScanConditions(path.TimestampConditions), "search_conditions": celScanConditions(path.SearchConditions),
			"seek_columns": path.SeekColumns, "residual_columns": path.ResidualColumns,
			"seekable_key_size": path.SeekableKeySize, "range_seekable_key_size": rangeSize,
			"declared_key_count": keyCount, "key_columns": path.KeyColumns,
			"stored_columns": path.StoredColumns, "null_filtered": nullFiltered,
		})
	}
	return out
}

func accessPathNullableBool(value *bool) interface{} {
	if value == nil {
		return nil
	}
	return *value
}

func celScanConditions(conditions []ScanCondition) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(conditions))
	for _, condition := range conditions {
		out = append(out, map[string]interface{}{
			"index": condition.Index, "expression": condition.Expression, "columns": condition.Columns,
		})
	}
	return out
}
