package examples

import (
	"fmt"
	"log"

	orm "github.com/medatechnology/simpleorm"
	"github.com/medatechnology/simpleorm/postgres"
)

// ExampleSimpleCTE demonstrates basic CTE usage with structured approach
func ExampleSimpleCTE() {
	config := postgres.PostgresConfig{
		Host:     "localhost",
		Port:     5432,
		DBName:   "mydb",
		User:     "postgres",
		Password: "password",
	}

	db, err := postgres.NewDatabase(config)
	if err != nil {
		log.Fatal(err)
	}

	// Example 1: Simple CTE - filter active users first
	query := &orm.ComplexQuery{
		CTEs: []orm.CommonTableExpression{
			{
				Name: "active_users",
				Query: &orm.ComplexQuery{
					Select: []string{"id", "name", "email", "created_at"},
					From:   "users",
					Where: &orm.Condition{
						Field:    "status",
						Operator: "=",
						Value:    "active",
					},
				},
			},
		},
		Select: []string{"*"},
		From:   "active_users",
		OrderBy: []string{"created_at DESC"},
		Limit:   10,
	}

	records, err := db.SelectManyComplex(query)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	// Generated SQL:
	// WITH active_users AS (SELECT id, name, email, created_at FROM users WHERE status = ?)
	// SELECT * FROM active_users ORDER BY created_at DESC LIMIT 10

	fmt.Printf("Found %d active users\n", len(records))
}

// ExampleMultipleCTEs demonstrates using multiple CTEs
func ExampleMultipleCTEs() {
	config := postgres.PostgresConfig{
		Host:     "localhost",
		Port:     5432,
		DBName:   "mydb",
		User:     "postgres",
		Password: "password",
	}

	db, err := postgres.NewDatabase(config)
	if err != nil {
		log.Fatal(err)
	}

	// Example 2: Multiple CTEs - calculate user stats and recent orders
	query := &orm.ComplexQuery{
		CTEs: []orm.CommonTableExpression{
			{
				Name: "user_stats",
				Query: &orm.ComplexQuery{
					Select: []string{
						"user_id",
						"COUNT(*) as order_count",
						"SUM(total) as total_spent",
					},
					From: "orders",
					Where: &orm.Condition{
						Field:    "status",
						Operator: "=",
						Value:    "completed",
					},
					GroupBy: []string{"user_id"},
				},
			},
			{
				Name: "recent_orders",
				Query: &orm.ComplexQuery{
					Select: []string{"user_id", "MAX(created_at) as last_order_date"},
					From:   "orders",
					GroupBy: []string{"user_id"},
				},
			},
		},
		Select: []string{
			"users.id",
			"users.name",
			"user_stats.order_count",
			"user_stats.total_spent",
			"recent_orders.last_order_date",
		},
		From: "users",
		Joins: []orm.Join{
			{
				Type:      orm.InnerJoin,
				Table:     "user_stats",
				Condition: "users.id = user_stats.user_id",
			},
			{
				Type:      orm.LeftJoin,
				Table:     "recent_orders",
				Condition: "users.id = recent_orders.user_id",
			},
		},
		OrderBy: []string{"user_stats.total_spent DESC"},
		Limit:   20,
	}

	records, err := db.SelectManyComplex(query)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	// Generated SQL:
	// WITH user_stats AS (
	//   SELECT user_id, COUNT(*) as order_count, SUM(total) as total_spent
	//   FROM orders WHERE status = ? GROUP BY user_id
	// ),
	// recent_orders AS (
	//   SELECT user_id, MAX(created_at) as last_order_date FROM orders GROUP BY user_id
	// )
	// SELECT users.id, users.name, user_stats.order_count,
	//        user_stats.total_spent, recent_orders.last_order_date
	// FROM users
	// INNER JOIN user_stats ON users.id = user_stats.user_id
	// LEFT JOIN recent_orders ON users.id = recent_orders.user_id
	// ORDER BY user_stats.total_spent DESC LIMIT 20

	fmt.Printf("Found %d users with stats\n", len(records))
	for _, record := range records {
		fmt.Printf("User: %s, Orders: %v, Spent: %v\n",
			record.Data["name"],
			record.Data["order_count"],
			record.Data["total_spent"])
	}
}

// ExampleRecursiveCTE demonstrates recursive CTE for hierarchical data
func ExampleRecursiveCTE() {
	config := postgres.PostgresConfig{
		Host:     "localhost",
		Port:     5432,
		DBName:   "mydb",
		User:     "postgres",
		Password: "password",
	}

	db, err := postgres.NewDatabase(config)
	if err != nil {
		log.Fatal(err)
	}

	// Example 3: Recursive CTE - organizational hierarchy
	// Note: For complex recursive CTEs, you might want to use RawSQL
	query := &orm.ComplexQuery{
		CTEs: []orm.CommonTableExpression{
			{
				Name:      "org_hierarchy",
				Recursive: true,
				RawSQL: `
					SELECT id, name, manager_id, 1 as level
					FROM employees
					WHERE manager_id IS NULL
					UNION ALL
					SELECT e.id, e.name, e.manager_id, oh.level + 1
					FROM employees e
					INNER JOIN org_hierarchy oh ON e.manager_id = oh.id
				`,
			},
		},
		Select: []string{"id", "name", "level"},
		From:   "org_hierarchy",
		OrderBy: []string{"level", "name"},
	}

	records, err := db.SelectManyComplex(query)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	// Generated SQL:
	// WITH RECURSIVE org_hierarchy AS (
	//   SELECT id, name, manager_id, 1 as level
	//   FROM employees
	//   WHERE manager_id IS NULL
	//   UNION ALL
	//   SELECT e.id, e.name, e.manager_id, oh.level + 1
	//   FROM employees e
	//   INNER JOIN org_hierarchy oh ON e.manager_id = oh.id
	// )
	// SELECT id, name, level FROM org_hierarchy ORDER BY level, name

	fmt.Printf("Organization hierarchy:\n")
	for _, record := range records {
		level := record.Data["level"]
		indent := ""
		if lvl, ok := level.(int64); ok {
			for i := int64(1); i < lvl; i++ {
				indent += "  "
			}
		}
		fmt.Printf("%s- %s (Level %v)\n", indent, record.Data["name"], level)
	}
}

// ExampleCTEWithAggregation demonstrates CTE with complex aggregations
func ExampleCTEWithAggregation() {
	config := postgres.PostgresConfig{
		Host:     "localhost",
		Port:     5432,
		DBName:   "mydb",
		User:     "postgres",
		Password: "password",
	}

	db, err := postgres.NewDatabase(config)
	if err != nil {
		log.Fatal(err)
	}

	// Example 4: CTE for monthly sales aggregation
	query := &orm.ComplexQuery{
		CTEs: []orm.CommonTableExpression{
			{
				Name: "monthly_sales",
				Columns: []string{"month", "total_revenue", "order_count"},
				RawSQL: `
					SELECT
						DATE_TRUNC('month', created_at) as month,
						SUM(total) as total_revenue,
						COUNT(*) as order_count
					FROM orders
					WHERE status = 'completed'
					GROUP BY DATE_TRUNC('month', created_at)
				`,
			},
		},
		Select: []string{
			"month",
			"total_revenue",
			"order_count",
			"total_revenue / order_count as avg_order_value",
		},
		From: "monthly_sales",
		Where: &orm.Condition{
			Field:    "total_revenue",
			Operator: ">",
			Value:    1000,
		},
		OrderBy: []string{"month DESC"},
		Limit:   12,
	}

	records, err := db.SelectManyComplex(query)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Monthly sales report:\n")
	for _, record := range records {
		fmt.Printf("Month: %v, Revenue: %v, Orders: %v, Avg: %v\n",
			record.Data["month"],
			record.Data["total_revenue"],
			record.Data["order_count"],
			record.Data["avg_order_value"])
	}
}

// ExampleNestedCTE demonstrates nested CTEs building on each other
func ExampleNestedCTE() {
	config := postgres.PostgresConfig{
		Host:     "localhost",
		Port:     5432,
		DBName:   "mydb",
		User:     "postgres",
		Password: "password",
	}

	db, err := postgres.NewDatabase(config)
	if err != nil {
		log.Fatal(err)
	}

	// Example 5: Nested CTEs - filter, aggregate, then rank
	query := &orm.ComplexQuery{
		CTEs: []orm.CommonTableExpression{
			{
				Name: "active_products",
				Query: &orm.ComplexQuery{
					Select: []string{"id", "name", "category_id", "price"},
					From:   "products",
					Where: &orm.Condition{
						Field:    "active",
						Operator: "=",
						Value:    true,
					},
				},
			},
			{
				Name: "product_sales",
				RawSQL: `
					SELECT
						p.id,
						p.name,
						p.category_id,
						COUNT(oi.id) as units_sold,
						SUM(oi.quantity * oi.price) as revenue
					FROM active_products p
					LEFT JOIN order_items oi ON p.id = oi.product_id
					GROUP BY p.id, p.name, p.category_id
				`,
			},
			{
				Name: "ranked_products",
				RawSQL: `
					SELECT
						*,
						ROW_NUMBER() OVER (PARTITION BY category_id ORDER BY revenue DESC) as rank
					FROM product_sales
				`,
			},
		},
		Select: []string{"*"},
		From:   "ranked_products",
		Where: &orm.Condition{
			Field:    "rank",
			Operator: "<=",
			Value:    5,
		},
		OrderBy: []string{"category_id", "rank"},
	}

	records, err := db.SelectManyComplex(query)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	// This gives us top 5 products per category
	fmt.Printf("Top products by category:\n")
	currentCategory := int64(-1)
	for _, record := range records {
		categoryID := record.Data["category_id"]
		if catID, ok := categoryID.(int64); ok && catID != currentCategory {
			currentCategory = catID
			fmt.Printf("\nCategory %d:\n", currentCategory)
		}
		fmt.Printf("  %d. %s - Revenue: %v, Units: %v\n",
			record.Data["rank"],
			record.Data["name"],
			record.Data["revenue"],
			record.Data["units_sold"])
	}
}

// ExampleCTERawString demonstrates using raw CTE string for backward compatibility
func ExampleCTERawString() {
	config := postgres.PostgresConfig{
		Host:     "localhost",
		Port:     5432,
		DBName:   "mydb",
		User:     "postgres",
		Password: "password",
	}

	db, err := postgres.NewDatabase(config)
	if err != nil {
		log.Fatal(err)
	}

	// Example 6: Using raw CTE string for very complex cases
	query := &orm.ComplexQuery{
		CTERaw: `WITH active_users AS (
			SELECT id, name, email FROM users WHERE status = 'active'
		)`,
		Select: []string{"*"},
		From:   "active_users",
		Limit:  10,
	}

	records, err := db.SelectManyComplex(query)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Found %d users using raw CTE\n", len(records))
}

// ExampleCTEMaterialization demonstrates CTE materialization hints (PostgreSQL 12+)
func ExampleCTEMaterialization() {
	config := postgres.PostgresConfig{
		Host:     "localhost",
		Port:     5432,
		DBName:   "mydb",
		User:     "postgres",
		Password: "password",
	}

	db, err := postgres.NewDatabase(config)
	if err != nil {
		log.Fatal(err)
	}

	// Example 7: Using materialization hints for performance
	// Use RawSQL for PostgreSQL-specific features
	query := &orm.ComplexQuery{
		CTEs: []orm.CommonTableExpression{
			{
				Name: "expensive_calculation",
				// Use MATERIALIZED hint to force materialization
				RawSQL: `
					SELECT
						user_id,
						COUNT(*) as action_count,
						array_agg(action_type) as actions
					FROM user_actions
					WHERE created_at >= CURRENT_DATE - INTERVAL '30 days'
					GROUP BY user_id
				`,
			},
		},
		Select: []string{
			"users.id",
			"users.name",
			"ec.action_count",
		},
		From: "users",
		Joins: []orm.Join{
			{
				Type:      orm.InnerJoin,
				Table:     "expensive_calculation",
				Alias:     "ec",
				Condition: "users.id = ec.user_id",
			},
		},
		Where: &orm.Condition{
			Field:    "ec.action_count",
			Operator: ">",
			Value:    10,
		},
		Limit: 100,
	}

	records, err := db.SelectManyComplex(query)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Found %d active users\n", len(records))
}
