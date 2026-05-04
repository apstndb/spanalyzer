package spanalyzer

import googlesql "github.com/goccy/go-googlesql"

type AnalyzerOption func(*analyzerConfig)

type analyzerConfig struct {
	productMode                           *googlesql.ProductMode
	strictNameResolution                  bool
	foldLiteralCast                       *bool
	pruneUnusedColumns                    *bool
	parseLocationRecordType               *googlesql.ParseLocationRecordType
	rawMaximumDevelopmentLanguageFeatures bool
}

func defaultAnalyzerConfig() analyzerConfig {
	productMode := googlesql.ProductModeProductExternal
	return analyzerConfig{
		productMode: &productMode,
	}
}

func WithProductMode(mode googlesql.ProductMode) AnalyzerOption {
	return func(config *analyzerConfig) {
		config.productMode = &mode
	}
}

func WithStrictNameResolution(enabled bool) AnalyzerOption {
	return func(config *analyzerConfig) {
		config.strictNameResolution = enabled
	}
}

func WithFoldLiteralCast(enabled bool) AnalyzerOption {
	return func(config *analyzerConfig) {
		config.foldLiteralCast = &enabled
	}
}

func WithPruneUnusedColumns(enabled bool) AnalyzerOption {
	return func(config *analyzerConfig) {
		config.pruneUnusedColumns = &enabled
	}
}

func WithParseLocationRecordType(recordType googlesql.ParseLocationRecordType) AnalyzerOption {
	return func(config *analyzerConfig) {
		config.parseLocationRecordType = &recordType
	}
}

func WithMaximumDevelopmentLanguageFeatures(enabled bool) AnalyzerOption {
	return func(config *analyzerConfig) {
		config.rawMaximumDevelopmentLanguageFeatures = enabled
	}
}
