package togoquexpr_test

import (
	"fmt"
	"log"

	"github.com/doug-martin/goqu/v9"
	"togoquexpr"
)

func Example() {
	// Create a goqu dataset
	ds := goqu.From("users")

	// Define which columns contain JSON data
	jsonColumns := []string{"meta", "settings"}

	// Example 1: Simple equality
	result, err := togoquexpr.AddWhereClause(ds, jsonColumns, "id = 1")
	if err != nil {
		log.Fatal(err)
	}
	sql, _, _ := result.ToSQL()
	fmt.Println("Simple equality:", sql)

	// Example 2: JSON field query
	result, err = togoquexpr.AddWhereClause(ds, jsonColumns, "meta.name = 'John'")
	if err != nil {
		log.Fatal(err)
	}
	sql, _, _ = result.ToSQL()
	fmt.Println("JSON field:", sql)

	// Example 3: JSON array access
	result, err = togoquexpr.AddWhereClause(ds, jsonColumns, "settings.preferences[0].theme = 'dark'")
	if err != nil {
		log.Fatal(err)
	}
	sql, _, _ = result.ToSQL()
	fmt.Println("JSON array:", sql)

	// Example 4: Complex query with AND/OR
	result, err = togoquexpr.AddWhereClause(ds, jsonColumns, "(status = 'active' OR status = 'pending') AND meta.verified = true")
	if err != nil {
		log.Fatal(err)
	}
	sql, _, _ = result.ToSQL()
	fmt.Println("Complex query:", sql)

	// Example 5: IN operator
	result, err = togoquexpr.AddWhereClause(ds, jsonColumns, "role IN ('admin', 'moderator')")
	if err != nil {
		log.Fatal(err)
	}
	sql, _, _ = result.ToSQL()
	fmt.Println("IN operator:", sql)

	// Example 6: LIKE operator
	result, err = togoquexpr.AddWhereClause(ds, jsonColumns, "email LIKE '%@example.com'")
	if err != nil {
		log.Fatal(err)
	}
	sql, _, _ = result.ToSQL()
	fmt.Println("LIKE operator:", sql)

	// Output:
	// Simple equality: SELECT * FROM "users" WHERE ("id" = 1)
	// JSON field: SELECT * FROM "users" WHERE JSONExtractRaw(meta, '$.name') = 'John'
	// JSON array: SELECT * FROM "users" WHERE JSONExtractRaw(settings, '$.preferences[0].theme') = 'dark'
	// Complex query: SELECT * FROM "users" WHERE ((("status" = 'active') OR ("status" = 'pending')) AND JSONExtractRaw(meta, '$.verified') = true)
	// IN operator: SELECT * FROM "users" WHERE ("role" IN ('admin', 'moderator'))
	// LIKE operator: SELECT * FROM "users" WHERE ("email" LIKE '%@example.com')
}