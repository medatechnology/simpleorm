package examples

import (
	"fmt"
	"log"

	orm "github.com/medatechnology/simpleorm"
	"github.com/medatechnology/simpleorm/postgres"
)

// ExampleComplexQueryBasic demonstrates basic complex query usage with custom SELECT fields
func ExampleComplexQueryBasic() {
	// Initialize database connection
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

	// Example 1: Simple complex query with custom SELECT fields
	query := &orm.ComplexQuery{
		Select: []string{"id", "name", "email", "created_at"},
		From:   "users",
		Where: &orm.Condition{
			Field:    "status",
			Operator: "=",
			Value:    "active",
		},
		OrderBy: []string{"created_at DESC"},
		Limit:   10,
	}

	records, err := db.SelectManyComplex(query)
	if err != nil {
		log.Printf("Error executing query: %v", err)
		return
	}

	fmt.Printf("Found %d active users\n", len(records))
	for _, record := range records {
		fmt.Printf("User: %v\n", record.Data)
	}
}

// ExampleComplexQueryWithJoins demonstrates complex query with JOINs
func ExampleComplexQueryWithJoins() {
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

	// Example 2: Query with LEFT JOIN
	query := &orm.ComplexQuery{
		Select: []string{
			"users.id",
			"users.name",
			"users.email",
			"profiles.bio",
			"profiles.avatar_url",
		},
		From: "users",
		Joins: []orm.Join{
			{
				Type:      orm.LeftJoin,
				Table:     "profiles",
				Condition: "users.id = profiles.user_id",
			},
		},
		Where: &orm.Condition{
			Field:    "users.status",
			Operator: "=",
			Value:    "active",
		},
		OrderBy: []string{"users.created_at DESC"},
		Limit:   20,
	}

	records, err := db.SelectManyComplex(query)
	if err != nil {
		log.Printf("Error executing query: %v", err)
		return
	}

	fmt.Printf("Found %d users with profiles\n", len(records))
	for _, record := range records {
		fmt.Printf("User: %s, Bio: %v\n", record.Data["name"], record.Data["bio"])
	}
}

// ExampleComplexQueryWithAggregation demonstrates GROUP BY and aggregate functions
func ExampleComplexQueryWithAggregation() {
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

	// Example 3: Query with GROUP BY and aggregate functions
	query := &orm.ComplexQuery{
		Select: []string{
			"users.id",
			"users.name",
			"COUNT(orders.id) as order_count",
			"SUM(orders.total) as total_spent",
			"AVG(orders.total) as avg_order_value",
		},
		From: "users",
		Joins: []orm.Join{
			{
				Type:      orm.LeftJoin,
				Table:     "orders",
				Condition: "users.id = orders.user_id",
			},
		},
		Where: &orm.Condition{
			Field:    "users.status",
			Operator: "=",
			Value:    "active",
		},
		GroupBy: []string{"users.id", "users.name"},
		Having:  "COUNT(orders.id) > 5",
		OrderBy: []string{"order_count DESC"},
		Limit:   10,
	}

	records, err := db.SelectManyComplex(query)
	if err != nil {
		log.Printf("Error executing query: %v", err)
		return
	}

	fmt.Printf("Found %d users with more than 5 orders\n", len(records))
	for _, record := range records {
		fmt.Printf("User: %s, Orders: %v, Total Spent: %v\n",
			record.Data["name"],
			record.Data["order_count"],
			record.Data["total_spent"])
	}
}

// ExampleComplexQueryMultipleJoins demonstrates multiple JOINs
func ExampleComplexQueryMultipleJoins() {
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

	// Example 4: Query with multiple JOINs
	query := &orm.ComplexQuery{
		Select: []string{
			"users.id",
			"users.name",
			"orders.order_number",
			"orders.total",
			"products.name as product_name",
			"order_items.quantity",
		},
		From: "users",
		Joins: []orm.Join{
			{
				Type:      orm.InnerJoin,
				Table:     "orders",
				Condition: "users.id = orders.user_id",
			},
			{
				Type:      orm.InnerJoin,
				Table:     "order_items",
				Condition: "orders.id = order_items.order_id",
			},
			{
				Type:      orm.InnerJoin,
				Table:     "products",
				Condition: "order_items.product_id = products.id",
			},
		},
		Where: &orm.Condition{
			Logic: "AND",
			Nested: []orm.Condition{
				{
					Field:    "users.status",
					Operator: "=",
					Value:    "active",
				},
				{
					Field:    "orders.status",
					Operator: "=",
					Value:    "completed",
				},
			},
		},
		OrderBy: []string{"orders.created_at DESC"},
		Limit:   50,
	}

	records, err := db.SelectManyComplex(query)
	if err != nil {
		log.Printf("Error executing query: %v", err)
		return
	}

	fmt.Printf("Found %d order items\n", len(records))
	for _, record := range records {
		fmt.Printf("User: %s, Order: %s, Product: %s, Qty: %v\n",
			record.Data["name"],
			record.Data["order_number"],
			record.Data["product_name"],
			record.Data["quantity"])
	}
}

// ExampleComplexQueryWithDistinct demonstrates DISTINCT usage
func ExampleComplexQueryWithDistinct() {
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

	// Example 5: Query with DISTINCT to get unique cities
	query := &orm.ComplexQuery{
		Select:   []string{"DISTINCT city", "country"},
		Distinct: true,
		From:     "users",
		Where: &orm.Condition{
			Field:    "status",
			Operator: "=",
			Value:    "active",
		},
		OrderBy: []string{"country", "city"},
	}

	records, err := db.SelectManyComplex(query)
	if err != nil {
		log.Printf("Error executing query: %v", err)
		return
	}

	fmt.Printf("Found %d unique cities\n", len(records))
	for _, record := range records {
		fmt.Printf("City: %s, Country: %s\n", record.Data["city"], record.Data["country"])
	}
}

// ExampleSelectOneComplex demonstrates SelectOneComplex usage
func ExampleSelectOneComplex() {
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

	// Example 6: Get a single user with their profile
	query := &orm.ComplexQuery{
		Select: []string{
			"users.*",
			"profiles.bio",
			"profiles.avatar_url",
			"profiles.location",
		},
		From: "users",
		Joins: []orm.Join{
			{
				Type:      orm.InnerJoin,
				Table:     "profiles",
				Condition: "users.id = profiles.user_id",
			},
		},
		Where: &orm.Condition{
			Field:    "users.id",
			Operator: "=",
			Value:    123,
		},
	}

	record, err := db.SelectOneComplex(query)
	if err != nil {
		log.Printf("Error executing query: %v", err)
		return
	}

	fmt.Printf("User found: %v\n", record.Data)
}

// ExampleComplexQueryWithNestedConditions demonstrates complex nested WHERE conditions
func ExampleComplexQueryWithNestedConditions() {
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

	// Example 7: Complex nested conditions
	// SELECT * FROM users
	// WHERE ((age > 18 AND country = 'USA') OR (status = 'premium' AND verified = true))
	// ORDER BY created_at DESC
	query := &orm.ComplexQuery{
		Select: []string{"*"},
		From:   "users",
		Where: &orm.Condition{
			Logic: "OR",
			Nested: []orm.Condition{
				{
					Logic: "AND",
					Nested: []orm.Condition{
						{
							Field:    "age",
							Operator: ">",
							Value:    18,
						},
						{
							Field:    "country",
							Operator: "=",
							Value:    "USA",
						},
					},
				},
				{
					Logic: "AND",
					Nested: []orm.Condition{
						{
							Field:    "status",
							Operator: "=",
							Value:    "premium",
						},
						{
							Field:    "verified",
							Operator: "=",
							Value:    true,
						},
					},
				},
			},
		},
		OrderBy: []string{"created_at DESC"},
		Limit:   25,
	}

	records, err := db.SelectManyComplex(query)
	if err != nil {
		log.Printf("Error executing query: %v", err)
		return
	}

	fmt.Printf("Found %d users matching complex criteria\n", len(records))
	for _, record := range records {
		fmt.Printf("User: %v\n", record.Data)
	}
}
