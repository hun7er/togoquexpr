package togoquexpr

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/xwb1989/sqlparser"
)

func AddWhereClause(ds *goqu.SelectDataset, jsonColumns []string, whereStr string) (*goqu.SelectDataset, error) {
	// Pre-process the where string to handle JSON array notation
	processedWhere := preprocessJsonPaths(whereStr)
	
	stmt, err := sqlparser.Parse("SELECT * FROM t WHERE " + processedWhere)
	if err != nil {
		return nil, fmt.Errorf("invalid query: %v", err)
	}

	sel, ok := stmt.(*sqlparser.Select)
	if !ok {
		return nil, fmt.Errorf("invalid query")
	}

	goquExpr, err := toGoquExpr(jsonColumns, sel.Where.Expr)
	if err != nil {
		return nil, fmt.Errorf("invalid query: %v", err)
	}

	return ds.Where(goquExpr), nil
}

// preprocessJsonPaths converts JSON array notation to a parseable format
// e.g., meta.alfa[0][1].gamma -> meta__alfa__0__1__gamma
func preprocessJsonPaths(whereStr string) string {
	// Pattern to match JSON paths with array indices, but not floats
	// Look for word followed by dots/brackets that aren't just numbers
	re := regexp.MustCompile(`(\w+)((?:\.\w+|\[[^\]]+\])+)`)
	
	return re.ReplaceAllStringFunc(whereStr, func(match string) string {
		// Check if this is a float (e.g., 100.5)
		if matched, _ := regexp.MatchString(`^\d+\.\d+$`, match); matched {
			return match // Don't process floats
		}
		
		// Replace dots and brackets with double underscores
		processed := strings.ReplaceAll(match, ".", "__")
		processed = strings.ReplaceAll(processed, "[", "__")
		processed = strings.ReplaceAll(processed, "]", "")
		return processed
	})
}

// reverseJsonPath converts the preprocessed path back to original format
func reverseJsonPath(processedPath string, jsonColumns []string) (string, string, bool) {
	// Check if this is a JSON column path
	for _, jsonCol := range jsonColumns {
		if strings.HasPrefix(processedPath, jsonCol+"__") {
			// Extract the path after the JSON column name
			pathPart := strings.TrimPrefix(processedPath, jsonCol+"__")
			parts := strings.Split(pathPart, "__")
			
			var result []string
			for _, part := range parts {
				// Check if part is a number (array index)
				if _, err := strconv.Atoi(part); err == nil {
					// It's an array index
					if len(result) > 0 {
						result[len(result)-1] += "[" + part + "]"
					}
				} else {
					// It's a field name
					result = append(result, part)
				}
			}
			
			return jsonCol, strings.Join(result, "."), true
		} else if processedPath == jsonCol {
			return jsonCol, "", true
		}
	}
	
	// Not a JSON path, could be a regular column
	return processedPath, "", false
}

func toGoquExpr(jsonColumns []string, expr sqlparser.Expr) (exp.Expression, error) {
	switch e := expr.(type) {
	case *sqlparser.ComparisonExpr:
		return handleComparison(jsonColumns, e)
	case *sqlparser.AndExpr:
		left, err := toGoquExpr(jsonColumns, e.Left)
		if err != nil {
			return nil, err
		}
		right, err := toGoquExpr(jsonColumns, e.Right)
		if err != nil {
			return nil, err
		}
		return goqu.And(left, right), nil
	case *sqlparser.OrExpr:
		left, err := toGoquExpr(jsonColumns, e.Left)
		if err != nil {
			return nil, err
		}
		right, err := toGoquExpr(jsonColumns, e.Right)
		if err != nil {
			return nil, err
		}
		return goqu.Or(left, right), nil
	case *sqlparser.ParenExpr:
		return toGoquExpr(jsonColumns, e.Expr)
	default:
		return nil, fmt.Errorf("unsupported expression type: %T", expr)
	}
}

func handleComparison(jsonColumns []string, expr *sqlparser.ComparisonExpr) (exp.Expression, error) {
	leftExpr, leftIsJson, err := extractExpression(jsonColumns, expr.Left)
	if err != nil {
		return nil, err
	}

	switch expr.Operator {
	case sqlparser.InStr:
		// Handle IN operator specially
		values, err := extractInValues(jsonColumns, expr.Right)
		if err != nil {
			return nil, err
		}
		if leftIsJson {
			// For JSON fields, we need to use literal SQL
			return goqu.L(fmt.Sprintf("%s IN ?", leftExpr), values), nil
		}
		return goqu.C(leftExpr.(string)).In(values...), nil
	default:
		rightVal, err := extractValue(jsonColumns, expr.Right)
		if err != nil {
			return nil, err
		}
		
		if leftIsJson {
			// For JSON fields, we need to use literal SQL
			switch expr.Operator {
			case sqlparser.EqualStr:
				return goqu.L(fmt.Sprintf("%s = ?", leftExpr), rightVal), nil
			case sqlparser.NotEqualStr:
				return goqu.L(fmt.Sprintf("%s != ?", leftExpr), rightVal), nil
			case sqlparser.LessThanStr:
				return goqu.L(fmt.Sprintf("%s < ?", leftExpr), rightVal), nil
			case sqlparser.GreaterThanStr:
				return goqu.L(fmt.Sprintf("%s > ?", leftExpr), rightVal), nil
			case sqlparser.LessEqualStr:
				return goqu.L(fmt.Sprintf("%s <= ?", leftExpr), rightVal), nil
			case sqlparser.GreaterEqualStr:
				return goqu.L(fmt.Sprintf("%s >= ?", leftExpr), rightVal), nil
			case sqlparser.LikeStr:
				return goqu.L(fmt.Sprintf("%s LIKE ?", leftExpr), rightVal), nil
			default:
				return nil, fmt.Errorf("unsupported operator: %s", expr.Operator)
			}
		}
		
		// For regular columns, use goqu's expression builder
		col := goqu.C(leftExpr.(string))
		switch expr.Operator {
		case sqlparser.EqualStr:
			return col.Eq(rightVal), nil
		case sqlparser.NotEqualStr:
			return col.Neq(rightVal), nil
		case sqlparser.LessThanStr:
			return col.Lt(rightVal), nil
		case sqlparser.GreaterThanStr:
			return col.Gt(rightVal), nil
		case sqlparser.LessEqualStr:
			return col.Lte(rightVal), nil
		case sqlparser.GreaterEqualStr:
			return col.Gte(rightVal), nil
		case sqlparser.LikeStr:
			pattern, ok := rightVal.(string)
			if !ok {
				return nil, fmt.Errorf("LIKE pattern must be a string")
			}
			return col.Like(pattern), nil
		default:
			return nil, fmt.Errorf("unsupported operator: %s", expr.Operator)
		}
	}
}

func extractExpression(jsonColumns []string, expr sqlparser.Expr) (interface{}, bool, error) {
	switch e := expr.(type) {
	case *sqlparser.ColName:
		colName := e.Name.String()
		
		// Check if it's a preprocessed JSON column path
		if jsonCol, path, isJson := reverseJsonPath(colName, jsonColumns); isJson {
			if path == "" {
				// Just the JSON column name without path
				return fmt.Sprintf("JSONExtractRaw(%s)", jsonCol), true, nil
			}
			// Convert path to ClickHouse JSON path format
			clickhousePath := convertToClickhousePath(path)
			return fmt.Sprintf("JSONExtractRaw(%s, %s)", jsonCol, clickhousePath), true, nil
		}
		
		// Regular column
		return colName, false, nil
		
	case *sqlparser.FuncExpr:
		// Functions are not allowed
		return "", false, fmt.Errorf("functions are not allowed")
		
	default:
		return "", false, fmt.Errorf("unsupported left-hand expression type: %T", expr)
	}
}

func extractValue(jsonColumns []string, expr sqlparser.Expr) (interface{}, error) {
	switch e := expr.(type) {
	case *sqlparser.ColName:
		// For right-hand side values, column names are not allowed to prevent injection
		return nil, fmt.Errorf("column names are not allowed as values")
		
	case *sqlparser.SQLVal:
		switch e.Type {
		case sqlparser.StrVal:
			return string(e.Val), nil
		case sqlparser.IntVal:
			val, err := strconv.ParseInt(string(e.Val), 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid integer value: %s", e.Val)
			}
			return val, nil
		case sqlparser.FloatVal:
			val, err := strconv.ParseFloat(string(e.Val), 64)
			if err != nil {
				return nil, fmt.Errorf("invalid float value: %s", e.Val)
			}
			return val, nil
		default:
			return nil, fmt.Errorf("unsupported value type: %v", e.Type)
		}
		
	case *sqlparser.NullVal:
		return nil, nil
		
	case *sqlparser.BoolVal:
		return bool(*e), nil
		
	case *sqlparser.FuncExpr:
		// Functions are not allowed
		return nil, fmt.Errorf("functions are not allowed")
		
	default:
		return nil, fmt.Errorf("unsupported expression type: %T", expr)
	}
}

func extractInValues(jsonColumns []string, expr sqlparser.Expr) ([]interface{}, error) {
	switch e := expr.(type) {
	case sqlparser.ValTuple:
		values := make([]interface{}, 0, len(e))
		for _, val := range e {
			v, err := extractValue(jsonColumns, val)
			if err != nil {
				return nil, err
			}
			values = append(values, v)
		}
		return values, nil
	case *sqlparser.ParenExpr:
		// Try to extract from parenthesized expression
		return extractInValues(jsonColumns, e.Expr)
	default:
		// Try direct value extraction for single value IN clauses
		val, err := extractValue(jsonColumns, expr)
		if err != nil {
			return nil, fmt.Errorf("IN clause requires a list of values")
		}
		return []interface{}{val}, nil
	}
}

func convertToClickhousePath(path string) string {
	// Convert JavaScript-style path to ClickHouse JSON path
	// alfa[0][1].gamma -> '$.alfa[0][1].gamma'
	return fmt.Sprintf("'$.%s'", path)
}