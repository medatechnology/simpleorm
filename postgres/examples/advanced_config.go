//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"log"
	"time"

	"github.com/medatechnology/simpleorm/postgres"
)

func main() {
	fmt.Println("=== Advanced PostgreSQL Configuration Examples ===\n")

	// Example 1: Using method chaining for fluent configuration
	fmt.Println("Example 1: Fluent Configuration")
	config1 := postgres.NewDefaultConfig().
		WithSSLMode("require").
		WithConnectionPool(50, 10, 10*time.Minute, 5*time.Minute).
		WithTimeouts(20*time.Second, 60*time.Second).
		WithApplicationName("my-awesome-app").
		WithSearchPath("public,custom_schema").
		WithTimezone("UTC").
		WithExtraParam("statement_timeout", "30000")

	fmt.Printf("Config: %s\n\n", config1.String())

	// Example 2: Creating configuration from environment-style settings
	fmt.Println("Example 2: Manual Configuration")
	config2 := &postgres.PostgresConfig{
		Host:            "db.example.com",
		Port:            5432,
		User:            "app_user",
		Password:        "secure_password",
		DBName:          "production_db",
		SSLMode:         "verify-full",
		MaxOpenConns:    100,
		MaxIdleConns:    20,
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 10 * time.Minute,
		ConnectTimeout:  15 * time.Second,
		QueryTimeout:    45 * time.Second,
		ApplicationName: "production-service",
		SearchPath:      "app_schema,public",
		Timezone:        "America/New_York",
	}

	// Validate the configuration
	if err := config2.Validate(); err != nil {
		log.Fatalf("Configuration validation failed: %v", err)
	}
	fmt.Println("Configuration validated successfully\n")

	// Example 3: Generate different DSN formats
	fmt.Println("Example 3: DSN Generation")

	// URL format
	urlDSN, err := config2.ToDSN()
	if err != nil {
		log.Fatalf("Failed to generate URL DSN: %v", err)
	}
	fmt.Printf("URL Format DSN:\n%s\n\n", urlDSN)

	// Simple key=value format
	simpleDSN, err := config2.ToSimpleDSN()
	if err != nil {
		log.Fatalf("Failed to generate simple DSN: %v", err)
	}
	fmt.Printf("Simple Format DSN:\n%s\n\n", simpleDSN)

	// Example 4: Parse existing DSN
	fmt.Println("Example 4: Parsing DSN")
	existingDSN := "postgres://testuser:testpass@localhost:5432/testdb?sslmode=disable&application_name=testapp"
	parsedConfig, err := postgres.ParseDSN(existingDSN)
	if err != nil {
		log.Fatalf("Failed to parse DSN: %v", err)
	}
	fmt.Printf("Parsed Config:\n")
	fmt.Printf("  Host: %s\n", parsedConfig.Host)
	fmt.Printf("  Port: %d\n", parsedConfig.Port)
	fmt.Printf("  User: %s\n", parsedConfig.User)
	fmt.Printf("  Database: %s\n", parsedConfig.DBName)
	fmt.Printf("  SSL Mode: %s\n", parsedConfig.SSLMode)
	fmt.Printf("  Application: %s\n\n", parsedConfig.ApplicationName)

	// Example 5: Parsing key=value DSN
	fmt.Println("Example 5: Parsing Key=Value DSN")
	kvDSN := "host=localhost port=5432 user=postgres password=secret dbname=mydb sslmode=require"
	kvConfig, err := postgres.ParseDSN(kvDSN)
	if err != nil {
		log.Fatalf("Failed to parse key=value DSN: %v", err)
	}
	fmt.Printf("Parsed from key=value: %s\n\n", kvConfig.String())

	// Example 6: Cloning and modifying configuration
	fmt.Println("Example 6: Configuration Cloning")
	original := postgres.NewConfig("localhost", 5432, "user", "pass", "db")
	original.WithSSLMode("require").WithApplicationName("original-app")

	// Clone and modify for a different environment
	devConfig := original.Clone()
	devConfig.WithSSLMode("disable").
		WithApplicationName("dev-app").
		WithConnectionPool(10, 2, 5*time.Minute, 2*time.Minute)

	fmt.Printf("Original: %s (SSL: %s, App: %s)\n",
		original.String(), original.SSLMode, original.ApplicationName)
	fmt.Printf("Dev Clone: %s (SSL: %s, App: %s, MaxOpen: %d)\n\n",
		devConfig.String(), devConfig.SSLMode, devConfig.ApplicationName, devConfig.MaxOpenConns)

	// Example 7: Connection pool optimization for different workloads
	fmt.Println("Example 7: Workload-Specific Configurations")

	// High-throughput API server
	apiConfig := postgres.NewDefaultConfig().
		WithConnectionPool(100, 25, 15*time.Minute, 5*time.Minute).
		WithTimeouts(5*time.Second, 30*time.Second).
		WithApplicationName("api-server")
	fmt.Printf("API Server Config: MaxOpen=%d, MaxIdle=%d, ConnectTimeout=%v\n",
		apiConfig.MaxOpenConns, apiConfig.MaxIdleConns, apiConfig.ConnectTimeout)

	// Background job processor
	jobConfig := postgres.NewDefaultConfig().
		WithConnectionPool(10, 2, 30*time.Minute, 10*time.Minute).
		WithTimeouts(30*time.Second, 5*time.Minute).
		WithApplicationName("job-processor")
	fmt.Printf("Job Processor Config: MaxOpen=%d, MaxIdle=%d, QueryTimeout=%v\n",
		jobConfig.MaxOpenConns, jobConfig.MaxIdleConns, jobConfig.QueryTimeout)

	// Analytics/reporting system
	analyticsConfig := postgres.NewDefaultConfig().
		WithConnectionPool(5, 1, 1*time.Hour, 30*time.Minute).
		WithTimeouts(1*time.Minute, 30*time.Minute).
		WithApplicationName("analytics")
	fmt.Printf("Analytics Config: MaxOpen=%d, MaxIdle=%d, QueryTimeout=%v\n\n",
		analyticsConfig.MaxOpenConns, analyticsConfig.MaxIdleConns, analyticsConfig.QueryTimeout)

	// Example 8: Using extra parameters
	fmt.Println("Example 8: Custom PostgreSQL Parameters")
	customConfig := postgres.NewDefaultConfig().
		WithExtraParam("statement_timeout", "60000").        // 60 second query timeout
		WithExtraParam("lock_timeout", "10000").             // 10 second lock timeout
		WithExtraParam("idle_in_transaction_session_timeout", "300000") // 5 minute idle transaction timeout

	fmt.Printf("Extra parameters set:\n")
	for key, value := range customConfig.ExtraParams {
		fmt.Printf("  %s = %s\n", key, value)
	}

	fmt.Println("\n=== Configuration Examples Complete ===")
}
