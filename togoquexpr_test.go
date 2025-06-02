package togoquexpr

import (
	"testing"

	"github.com/doug-martin/goqu/v9"
)

func TestAddWhereClause(t *testing.T) {
	dialect := goqu.Dialect("default")
	
	tests := []struct {
		name        string
		jsonColumns []string
		whereStr    string
		expected    string
		wantErr     bool
	}{
		{
			name:        "simple equality",
			jsonColumns: []string{},
			whereStr:    "id = 1",
			expected:    `SELECT * FROM "test" WHERE ("id" = 1)`,
		},
		{
			name:        "simple inequality",
			jsonColumns: []string{},
			whereStr:    "status != 'active'",
			expected:    `SELECT * FROM "test" WHERE ("status" != 'active')`,
		},
		{
			name:        "greater than",
			jsonColumns: []string{},
			whereStr:    "age > 18",
			expected:    `SELECT * FROM "test" WHERE ("age" > 18)`,
		},
		{
			name:        "less than",
			jsonColumns: []string{},
			whereStr:    "price < 100.5",
			expected:    `SELECT * FROM "test" WHERE ("price" < 100.5)`,
		},
		{
			name:        "greater or equal",
			jsonColumns: []string{},
			whereStr:    "score >= 90",
			expected:    `SELECT * FROM "test" WHERE ("score" >= 90)`,
		},
		{
			name:        "less or equal",
			jsonColumns: []string{},
			whereStr:    "count <= 10",
			expected:    `SELECT * FROM "test" WHERE ("count" <= 10)`,
		},
		{
			name:        "LIKE operator",
			jsonColumns: []string{},
			whereStr:    "name LIKE '%john%'",
			expected:    `SELECT * FROM "test" WHERE ("name" LIKE '%john%')`,
		},
		{
			name:        "IN operator",
			jsonColumns: []string{},
			whereStr:    "status IN ('active', 'pending', 'completed')",
			expected:    `SELECT * FROM "test" WHERE ("status" IN ('active', 'pending', 'completed'))`,
		},
		{
			name:        "AND condition",
			jsonColumns: []string{},
			whereStr:    "age > 18 AND status = 'active'",
			expected:    `SELECT * FROM "test" WHERE (("age" > 18) AND ("status" = 'active'))`,
		},
		{
			name:        "OR condition",
			jsonColumns: []string{},
			whereStr:    "status = 'active' OR status = 'pending'",
			expected:    `SELECT * FROM "test" WHERE (("status" = 'active') OR ("status" = 'pending'))`,
		},
		{
			name:        "JSON field simple",
			jsonColumns: []string{"meta"},
			whereStr:    "meta.name = 'test'",
			expected:    `SELECT * FROM "test" WHERE JSONExtractRaw(meta, '$.name') = 'test'`,
		},
		{
			name:        "JSON field with array",
			jsonColumns: []string{"meta"},
			whereStr:    "meta.alfa[0][1].gamma = 'value'",
			expected:    `SELECT * FROM "test" WHERE JSONExtractRaw(meta, '$.alfa[0][1].gamma') = 'value'`,
		},
		{
			name:        "JSON field with complex path",
			jsonColumns: []string{"data"},
			whereStr:    "data.items[2].properties.name = 'test'",
			expected:    `SELECT * FROM "test" WHERE JSONExtractRaw(data, '$.items[2].properties.name') = 'test'`,
		},
		{
			name:        "Multiple JSON columns",
			jsonColumns: []string{"meta", "data"},
			whereStr:    "meta.status = 'active' AND data.count > 10",
			expected:    `SELECT * FROM "test" WHERE (JSONExtractRaw(meta, '$.status') = 'active' AND JSONExtractRaw(data, '$.count') > 10)`,
		},
		{
			name:        "Parentheses",
			jsonColumns: []string{},
			whereStr:    "(age > 18 AND status = 'active') OR role = 'admin'",
			expected:    `SELECT * FROM "test" WHERE ((("age" > 18) AND ("status" = 'active')) OR ("role" = 'admin'))`,
		},
		{
			name:        "Function call rejected",
			jsonColumns: []string{},
			whereStr:    "LENGTH(name) > 5",
			wantErr:     true,
		},
		{
			name:        "Arithmetic operations rejected",
			jsonColumns: []string{},
			whereStr:    "price + tax > 100",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ds := dialect.From("test")
			result, err := AddWhereClause(ds, tt.jsonColumns, tt.whereStr)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			
			sql, _, err := result.ToSQL()
			if err != nil {
				t.Errorf("failed to generate SQL: %v", err)
				return
			}
			
			if sql != tt.expected {
				t.Errorf("expected:\n%s\ngot:\n%s", tt.expected, sql)
			}
		})
	}
}