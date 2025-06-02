# togoquexpr

A Go library that converts SQL WHERE clauses to goqu expressions with support for ClickHouse JSON fields.

## Features

- Supports comparison operators: `=`, `!=`, `>`, `<`, `>=`, `<=`
- Supports `LIKE` and `IN` operators
- Supports logical operators: `AND`, `OR`
- Supports ClickHouse JSON fields with array notation (e.g., `meta.field[0][1].value`)
- Prevents SQL injection by validating input and disallowing:
  - Function calls
  - Arithmetic operations
  - Column names as values in comparisons

## Installation

```bash
go get github.com/yourusername/togoquexpr
```

## Usage

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/doug-martin/goqu/v9"
    "togoquexpr"
)

func main() {
    // Create a goqu dataset
    ds := goqu.From("users")
    
    // Define which columns contain JSON data
    jsonColumns := []string{"meta", "settings"}
    
    // Convert WHERE clause to goqu expression
    result, err := togoquexpr.AddWhereClause(ds, jsonColumns, "meta.name = 'John' AND status = 'active'")
    if err != nil {
        log.Fatal(err)
    }
    
    // Generate SQL
    sql, _, _ := result.ToSQL()
    fmt.Println(sql)
    // Output: SELECT * FROM "users" WHERE (JSONExtractRaw(meta, '$.name') = 'John' AND ("status" = 'active'))
}
```

## Examples

### Simple comparisons
```go
AddWhereClause(ds, []string{}, "age > 18")
// SELECT * FROM "table" WHERE ("age" > 18)
```

### JSON field access
```go
AddWhereClause(ds, []string{"meta"}, "meta.email = 'user@example.com'")
// SELECT * FROM "table" WHERE JSONExtractRaw(meta, '$.email') = 'user@example.com'
```

### JSON array access
```go
AddWhereClause(ds, []string{"data"}, "data.items[0].name = 'test'")
// SELECT * FROM "table" WHERE JSONExtractRaw(data, '$.items[0].name') = 'test'
```

### IN operator
```go
AddWhereClause(ds, []string{}, "status IN ('active', 'pending')")
// SELECT * FROM "table" WHERE ("status" IN ('active', 'pending'))
```

### LIKE operator
```go
AddWhereClause(ds, []string{}, "name LIKE '%john%'")
// SELECT * FROM "table" WHERE ("name" LIKE '%john%')
```

## Security

The library includes several security measures:

1. **No function calls allowed**: Prevents execution of SQL functions
2. **No arithmetic operations**: Prevents complex expressions that could be exploited
3. **No column names as values**: Right-hand side of comparisons must be literal values
4. **Input validation**: Uses sqlparser to validate the SQL syntax

## License

MIT