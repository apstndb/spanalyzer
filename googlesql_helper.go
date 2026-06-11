package spanalyzer

import (
	"bytes"
	"fmt"
	"sync"

	googlesql "github.com/goccy/go-googlesql"
)

var initGoogleSQLOnce sync.Once
var initGoogleSQLErr error

func InitGoogleSQL() error {
	initGoogleSQLOnce.Do(func() {
		initGoogleSQLErr = googlesql.Init()
	})
	return initGoogleSQLErr
}

type GoogleSQLHelper struct {
	Catalog     *googlesql.SimpleCatalog
	Options     *googlesql.AnalyzerOptions
	TypeFactory *googlesql.TypeFactory
}

func (h *GoogleSQLHelper) AnalyzeStatement(sql string) (*googlesql.AnalyzerOutput, error) {
	sql = normalizeSpannerFunctionNamedArgs(sql)
	return googlesql.AnalyzeStatement(sql, h.Options, h.Catalog, h.TypeFactory)
}

func (h *GoogleSQLHelper) AnalyzeExpression(sql string) (*googlesql.AnalyzerOutput, error) {
	sql = normalizeSpannerFunctionNamedArgs(sql)
	return googlesql.AnalyzeExpression(sql, h.Options, h.Catalog, h.TypeFactory)
}

func (h *GoogleSQLHelper) Parse(sqlMode, sql string) (*googlesql.ParserOutput, error) {
	parserOptions, err := h.Options.GetParserOptions()
	if err != nil {
		return nil, err
	}
	switch sqlMode {
	case "query":
		return googlesql.ParseStatement(sql, parserOptions)
	case "expression":
		return googlesql.ParseExpression(sql, parserOptions)
	default:
		return nil, fmt.Errorf("unsupported sql mode %q", sqlMode)
	}
}

func (h *GoogleSQLHelper) ParseDebugString(sqlMode, sql string) (string, error) {
	out, err := h.Parse(sqlMode, sql)
	if err != nil {
		return "", err
	}
	node, err := out.Node()
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := writeASTDebugString(&buf, node, 0); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (h *GoogleSQLHelper) Unparse(sqlMode, sql string) (string, error) {
	out, err := h.Parse(sqlMode, sql)
	if err != nil {
		return "", err
	}
	node, err := out.Node()
	if err != nil {
		return "", err
	}
	return googlesql.Unparse(node)
}

func (h *GoogleSQLHelper) ResolvedASTDebugString(sqlMode, sql string) (string, error) {
	var out *googlesql.AnalyzerOutput
	var err error
	switch sqlMode {
	case "query":
		out, err = h.AnalyzeStatement(sql)
	case "expression":
		out, err = h.AnalyzeExpression(sql)
	default:
		return "", fmt.Errorf("unsupported sql mode %q", sqlMode)
	}
	if err != nil {
		return "", err
	}
	node, err := out.ResolvedNode()
	if err != nil {
		return "", err
	}
	return node.DebugString()
}

func writeASTDebugString(buf *bytes.Buffer, node googlesql.ASTNode, depth int) error {
	if node == nil {
		return nil
	}
	for i := 0; i < depth; i++ {
		buf.WriteString("  ")
	}
	line, err := node.SingleNodeDebugString()
	if err != nil {
		return err
	}
	buf.WriteString(line)
	buf.WriteByte('\n')
	n, err := node.NumChildren()
	if err != nil {
		return err
	}
	for i := int32(0); i < n; i++ {
		child, err := node.Child(i)
		if err != nil {
			return err
		}
		if err := writeASTDebugString(buf, child, depth+1); err != nil {
			return err
		}
	}
	return nil
}
