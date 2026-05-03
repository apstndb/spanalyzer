package spanalyzer

import (
	"strings"

	"github.com/cloudspannerecosystem/memefish"
	"github.com/cloudspannerecosystem/memefish/ast"
)

var spannerFunctionArgNames = map[string][]string{
	"AI.CLASSIFY":        {"prompt", "categories"},
	"AI.IF":              {"prompt"},
	"AI.SCORE":           {"prompt"},
	"SCORE":              {"tokens", "query", "dialect", "language_tag", "enhance_query", "dictionary", "options"},
	"SCORE_NGRAMS":       {"tokens", "query", "language_tag", "algorithm", "array_aggregator"},
	"SEARCH":             {"tokens", "query", "dialect", "language_tag", "enhance_query", "dictionary"},
	"SEARCH_NGRAMS":      {"tokens", "query", "language_tag", "min_ngrams", "min_ngrams_percent"},
	"SEARCH_SUBSTRING":   {"tokens", "query", "language_tag", "relative_search_type"},
	"SNIPPET":            {"content", "query", "language_tag", "enhance_query", "dictionary", "max_snippet_width", "max_snippets", "content_type"},
	"TOKENIZE_FULLTEXT":  {"input", "language_tag", "content_type", "token_category", "remove_diacritics"},
	"TOKENIZE_NGRAMS":    {"input", "ngram_size_min", "ngram_size_max", "remove_diacritics"},
	"TOKENIZE_NUMBER":    {"input", "comparison_type", "algorithm", "min", "max", "granularity", "tree_base", "precision"},
	"TOKENIZE_SUBSTRING": {"input", "language_tag", "ngram_size_min", "ngram_size_max", "relative_search_types", "content_type", "short_tokens_only_for_anchors", "remove_diacritics"},
}

var spannerFunctionPathAliases = map[string]string{
	"AI.CLASSIFY": "AI_CLASSIFY",
	"AI.IF":       "AI_IF",
	"AI.SCORE":    "AI_SCORE",
}

func normalizeSpannerFunctionNamedArgs(sql string) string {
	stmt, err := memefish.ParseQuery("", sql)
	if err == nil {
		if normalizeSpannerFunctionCalls(stmt) {
			return stmt.SQL()
		}
		return sql
	}

	expr, err := memefish.ParseExpr("", sql)
	if err != nil {
		return sql
	}
	if normalizeSpannerFunctionCalls(expr) {
		return expr.SQL()
	}
	return sql
}

func normalizeSpannerFunctionCalls(node ast.Node) bool {
	changed := false
	ast.Inspect(node, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		if normalizeFunctionCallNamedArgs(call) {
			changed = true
		}
		if normalizeFunctionCallPath(call) {
			changed = true
		}
		return true
	})
	return changed
}

func normalizeFunctionCallNamedArgs(call *ast.CallExpr) bool {
	if len(call.NamedArgs) == 0 {
		return false
	}
	argNames, ok := spannerFunctionArgNames[functionPathKey(call.Func)]
	if !ok {
		return false
	}
	named := map[string]ast.Expr{}
	maxNamedIndex := -1
	for _, arg := range call.NamedArgs {
		index := indexStringFold(argNames, arg.Name.Name)
		if index < 0 {
			return false
		}
		named[strings.ToLower(arg.Name.Name)] = arg.Value
		if index > maxNamedIndex {
			maxNamedIndex = index
		}
	}
	if maxNamedIndex < len(call.Args) {
		return false
	}
	args := append([]ast.Arg(nil), call.Args...)
	for i := len(args); i <= maxNamedIndex; i++ {
		if expr, ok := named[strings.ToLower(argNames[i])]; ok {
			args = append(args, &ast.ExprArg{Expr: expr})
			continue
		}
		args = append(args, &ast.ExprArg{Expr: &ast.NullLiteral{}})
	}
	call.Args = args
	call.NamedArgs = nil
	return true
}

func normalizeFunctionCallPath(call *ast.CallExpr) bool {
	alias, ok := spannerFunctionPathAliases[functionPathKey(call.Func)]
	if !ok {
		return false
	}
	call.Func = &ast.Path{Idents: []*ast.Ident{{Name: alias}}}
	return true
}

func functionPathKey(path *ast.Path) string {
	parts := make([]string, 0, len(path.Idents))
	for _, ident := range path.Idents {
		parts = append(parts, strings.ToUpper(ident.Name))
	}
	if len(parts) > 1 && parts[0] == "SAFE" {
		parts = parts[1:]
	}
	return strings.Join(parts, ".")
}

func indexStringFold(values []string, target string) int {
	for i, value := range values {
		if strings.EqualFold(value, target) {
			return i
		}
	}
	return -1
}
