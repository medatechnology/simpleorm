package postgres

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

// Default configuration values
const (
	DefaultHost                = "localhost"
	DefaultPort                = 5432
	DefaultSSLMode             = "disable"
	DefaultConnectTimeout      = 10 * time.Second
	DefaultQueryTimeout        = 30 * time.Second
	DefaultMaxOpenConns        = 25
	DefaultMaxIdleConns        = 5
	DefaultConnMaxLifetime     = 5 * time.Minute
	DefaultConnMaxIdleTime     = 10 * time.Minute
	DefaultApplicationName     = "simpleorm"
)

// PostgresConfig holds the configuration for PostgreSQL database connection
type PostgresConfig struct {
	// Connection parameters
	Host     string // Database host (default: "localhost")
	Port     int    // Database port (default: 5432)
	User     string // Database user (required)
	Password string // Database password (required)
	DBName   string // Database name (required)
	SSLMode  string // SSL mode: "disable", "require", "verify-ca", "verify-full" (default: "disable")

	// Connection pooling
	MaxOpenConns    int           // Maximum number of open connections (default: 25)
	MaxIdleConns    int           // Maximum number of idle connections (default: 5)
	ConnMaxLifetime time.Duration // Maximum lifetime of a connection (default: 5 minutes)
	ConnMaxIdleTime time.Duration // Maximum idle time of a connection (default: 10 minutes)

	// Timeouts
	ConnectTimeout time.Duration // Connection timeout (default: 10 seconds)
	QueryTimeout   time.Duration // Query execution timeout (default: 30 seconds)

	// Additional parameters
	ApplicationName string            // Application name for logging (default: "simpleorm")
	SearchPath      string            // Schema search path (optional)
	Timezone        string            // Timezone (optional, e.g., "UTC")
	ExtraParams     map[string]string // Additional connection parameters (optional)
}

// NewDefaultConfig creates a new PostgresConfig with default values
func NewDefaultConfig() *PostgresConfig {
	return &PostgresConfig{
		Host:            DefaultHost,
		Port:            DefaultPort,
		SSLMode:         DefaultSSLMode,
		MaxOpenConns:    DefaultMaxOpenConns,
		MaxIdleConns:    DefaultMaxIdleConns,
		ConnMaxLifetime: DefaultConnMaxLifetime,
		ConnMaxIdleTime: DefaultConnMaxIdleTime,
		ConnectTimeout:  DefaultConnectTimeout,
		QueryTimeout:    DefaultQueryTimeout,
		ApplicationName: DefaultApplicationName,
		ExtraParams:     make(map[string]string),
	}
}

// NewConfig creates a new PostgresConfig with the specified database credentials
// and applies default values for other settings
func NewConfig(host string, port int, user, password, dbName string) *PostgresConfig {
	config := NewDefaultConfig()
	config.Host = host
	config.Port = port
	config.User = user
	config.Password = password
	config.DBName = dbName
	return config
}

// Validate checks if the configuration is valid
func (c *PostgresConfig) Validate() error {
	if c.User == "" {
		return fmt.Errorf("%w: user is required", ErrPostgresInvalidConfig)
	}
	if c.DBName == "" {
		return fmt.Errorf("%w: database name is required", ErrPostgresInvalidConfig)
	}
	if c.Host == "" {
		c.Host = DefaultHost
	}
	if c.Port <= 0 || c.Port > 65535 {
		c.Port = DefaultPort
	}
	if c.SSLMode == "" {
		c.SSLMode = DefaultSSLMode
	}
	if c.MaxOpenConns <= 0 {
		c.MaxOpenConns = DefaultMaxOpenConns
	}
	if c.MaxIdleConns <= 0 {
		c.MaxIdleConns = DefaultMaxIdleConns
	}
	if c.MaxIdleConns > c.MaxOpenConns {
		c.MaxIdleConns = c.MaxOpenConns
	}
	if c.ConnMaxLifetime <= 0 {
		c.ConnMaxLifetime = DefaultConnMaxLifetime
	}
	if c.ConnMaxIdleTime <= 0 {
		c.ConnMaxIdleTime = DefaultConnMaxIdleTime
	}
	if c.ConnectTimeout <= 0 {
		c.ConnectTimeout = DefaultConnectTimeout
	}
	if c.QueryTimeout <= 0 {
		c.QueryTimeout = DefaultQueryTimeout
	}
	if c.ApplicationName == "" {
		c.ApplicationName = DefaultApplicationName
	}

	// Validate SSL mode
	validSSLModes := map[string]bool{
		"disable":     true,
		"require":     true,
		"verify-ca":   true,
		"verify-full": true,
		"prefer":      true,
		"allow":       true,
	}
	if !validSSLModes[c.SSLMode] {
		return fmt.Errorf("%w: invalid SSL mode '%s'", ErrPostgresInvalidConfig, c.SSLMode)
	}

	return nil
}

// ToDSN converts the PostgresConfig to a Data Source Name (DSN) connection string
// Format: postgres://user:password@host:port/dbname?param1=value1&param2=value2
func (c *PostgresConfig) ToDSN() (string, error) {
	if err := c.Validate(); err != nil {
		return "", err
	}

	// Build the base DSN
	userInfo := url.UserPassword(c.User, c.Password)
	hostPort := fmt.Sprintf("%s:%d", c.Host, c.Port)

	u := &url.URL{
		Scheme: "postgres",
		User:   userInfo,
		Host:   hostPort,
		Path:   c.DBName,
	}

	// Build query parameters
	params := url.Values{}

	// SSL mode
	params.Set("sslmode", c.SSLMode)

	// Connect timeout (in seconds)
	if c.ConnectTimeout > 0 {
		params.Set("connect_timeout", fmt.Sprintf("%d", int(c.ConnectTimeout.Seconds())))
	}

	// Application name
	if c.ApplicationName != "" {
		params.Set("application_name", c.ApplicationName)
	}

	// Search path
	if c.SearchPath != "" {
		params.Set("search_path", c.SearchPath)
	}

	// Timezone
	if c.Timezone != "" {
		params.Set("timezone", c.Timezone)
	}

	// Add extra parameters
	for key, value := range c.ExtraParams {
		params.Set(key, value)
	}

	u.RawQuery = params.Encode()

	return u.String(), nil
}

// ToSimpleDSN converts the PostgresConfig to a simple DSN connection string
// Format: host=localhost port=5432 user=postgres password=secret dbname=mydb sslmode=disable
func (c *PostgresConfig) ToSimpleDSN() (string, error) {
	if err := c.Validate(); err != nil {
		return "", err
	}

	var parts []string

	parts = append(parts, fmt.Sprintf("host=%s", c.Host))
	parts = append(parts, fmt.Sprintf("port=%d", c.Port))
	parts = append(parts, fmt.Sprintf("user=%s", c.User))
	parts = append(parts, fmt.Sprintf("password=%s", c.Password))
	parts = append(parts, fmt.Sprintf("dbname=%s", c.DBName))
	parts = append(parts, fmt.Sprintf("sslmode=%s", c.SSLMode))

	if c.ConnectTimeout > 0 {
		parts = append(parts, fmt.Sprintf("connect_timeout=%d", int(c.ConnectTimeout.Seconds())))
	}

	if c.ApplicationName != "" {
		parts = append(parts, fmt.Sprintf("application_name=%s", c.ApplicationName))
	}

	if c.SearchPath != "" {
		parts = append(parts, fmt.Sprintf("search_path=%s", c.SearchPath))
	}

	if c.Timezone != "" {
		parts = append(parts, fmt.Sprintf("timezone=%s", c.Timezone))
	}

	// Add extra parameters
	for key, value := range c.ExtraParams {
		parts = append(parts, fmt.Sprintf("%s=%s", key, value))
	}

	return strings.Join(parts, " "), nil
}

// Clone creates a deep copy of the PostgresConfig
func (c *PostgresConfig) Clone() *PostgresConfig {
	clone := &PostgresConfig{
		Host:            c.Host,
		Port:            c.Port,
		User:            c.User,
		Password:        c.Password,
		DBName:          c.DBName,
		SSLMode:         c.SSLMode,
		MaxOpenConns:    c.MaxOpenConns,
		MaxIdleConns:    c.MaxIdleConns,
		ConnMaxLifetime: c.ConnMaxLifetime,
		ConnMaxIdleTime: c.ConnMaxIdleTime,
		ConnectTimeout:  c.ConnectTimeout,
		QueryTimeout:    c.QueryTimeout,
		ApplicationName: c.ApplicationName,
		SearchPath:      c.SearchPath,
		Timezone:        c.Timezone,
		ExtraParams:     make(map[string]string),
	}

	// Copy extra params
	for key, value := range c.ExtraParams {
		clone.ExtraParams[key] = value
	}

	return clone
}

// WithSSLMode sets the SSL mode and returns the config for method chaining
func (c *PostgresConfig) WithSSLMode(mode string) *PostgresConfig {
	c.SSLMode = mode
	return c
}

// WithConnectionPool sets the connection pool parameters and returns the config for method chaining
func (c *PostgresConfig) WithConnectionPool(maxOpen, maxIdle int, maxLifetime, maxIdleTime time.Duration) *PostgresConfig {
	c.MaxOpenConns = maxOpen
	c.MaxIdleConns = maxIdle
	c.ConnMaxLifetime = maxLifetime
	c.ConnMaxIdleTime = maxIdleTime
	return c
}

// WithTimeouts sets the timeout parameters and returns the config for method chaining
func (c *PostgresConfig) WithTimeouts(connectTimeout, queryTimeout time.Duration) *PostgresConfig {
	c.ConnectTimeout = connectTimeout
	c.QueryTimeout = queryTimeout
	return c
}

// WithApplicationName sets the application name and returns the config for method chaining
func (c *PostgresConfig) WithApplicationName(name string) *PostgresConfig {
	c.ApplicationName = name
	return c
}

// WithSearchPath sets the schema search path and returns the config for method chaining
func (c *PostgresConfig) WithSearchPath(path string) *PostgresConfig {
	c.SearchPath = path
	return c
}

// WithTimezone sets the timezone and returns the config for method chaining
func (c *PostgresConfig) WithTimezone(tz string) *PostgresConfig {
	c.Timezone = tz
	return c
}

// WithExtraParam adds an extra connection parameter and returns the config for method chaining
func (c *PostgresConfig) WithExtraParam(key, value string) *PostgresConfig {
	if c.ExtraParams == nil {
		c.ExtraParams = make(map[string]string)
	}
	c.ExtraParams[key] = value
	return c
}

// String returns a safe string representation of the config (without password)
func (c *PostgresConfig) String() string {
	return fmt.Sprintf("PostgreSQL{host=%s, port=%d, user=%s, dbname=%s, sslmode=%s}",
		c.Host, c.Port, c.User, c.DBName, c.SSLMode)
}

// ParseDSN parses a PostgreSQL DSN connection string and returns a PostgresConfig
// Supports both URL format and key=value format
func ParseDSN(dsn string) (*PostgresConfig, error) {
	config := NewDefaultConfig()

	// Try to parse as URL first
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		u, err := url.Parse(dsn)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrPostgresInvalidDSN, err)
		}

		// Extract user and password
		if u.User != nil {
			config.User = u.User.Username()
			if password, ok := u.User.Password(); ok {
				config.Password = password
			}
		}

		// Extract host and port
		if u.Host != "" {
			parts := strings.Split(u.Host, ":")
			config.Host = parts[0]
			if len(parts) > 1 {
				var port int
				fmt.Sscanf(parts[1], "%d", &port)
				if port > 0 {
					config.Port = port
				}
			}
		}

		// Extract database name
		config.DBName = strings.TrimPrefix(u.Path, "/")

		// Parse query parameters
		params := u.Query()
		if sslmode := params.Get("sslmode"); sslmode != "" {
			config.SSLMode = sslmode
		}
		if appName := params.Get("application_name"); appName != "" {
			config.ApplicationName = appName
		}
		if searchPath := params.Get("search_path"); searchPath != "" {
			config.SearchPath = searchPath
		}
		if tz := params.Get("timezone"); tz != "" {
			config.Timezone = tz
		}
		if timeout := params.Get("connect_timeout"); timeout != "" {
			var seconds int
			fmt.Sscanf(timeout, "%d", &seconds)
			if seconds > 0 {
				config.ConnectTimeout = time.Duration(seconds) * time.Second
			}
		}
	} else {
		// Parse key=value format
		pairs := strings.Split(dsn, " ")
		for _, pair := range pairs {
			parts := strings.SplitN(pair, "=", 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			switch key {
			case "host":
				config.Host = value
			case "port":
				var port int
				fmt.Sscanf(value, "%d", &port)
				if port > 0 {
					config.Port = port
				}
			case "user":
				config.User = value
			case "password":
				config.Password = value
			case "dbname":
				config.DBName = value
			case "sslmode":
				config.SSLMode = value
			case "application_name":
				config.ApplicationName = value
			case "search_path":
				config.SearchPath = value
			case "timezone":
				config.Timezone = value
			case "connect_timeout":
				var seconds int
				fmt.Sscanf(value, "%d", &seconds)
				if seconds > 0 {
					config.ConnectTimeout = time.Duration(seconds) * time.Second
				}
			default:
				// Store unknown parameters as extra params
				config.ExtraParams[key] = value
			}
		}
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}
