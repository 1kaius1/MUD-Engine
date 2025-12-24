// File: internal/config/config.go
// MUD Engine - Configuration Management

package config

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

// Config holds all configuration for the MUD server
type Config struct {
	// Server settings
	ServerName    string
	ServerVersion string
	ServerHost    string // Host/IP to bind to (empty string = all interfaces, "localhost" = local only)
	ServerPort    int
	
	// Database settings
	DBType           string // "sqlite" or "postgres"
	DBHost           string // For PostgreSQL
	DBPort           int    // For PostgreSQL
	DBName           string // Database name or file path for SQLite
	DBUser           string // For PostgreSQL
	DBPassword       string // For PostgreSQL
	DBMaxConnections int
	DBMaxIdleConns   int
	
	// Redis settings (for future use)
	RedisEnabled bool
	RedisHost    string
	RedisPort    int
	RedisDB      int
	
	// Server behavior
	MaxPlayers           int
	ShutdownTimeoutSecs  int
	ReconnectAttempts    int
	SessionTimeoutMins   int
	
	// TLS settings (for future use)
	TLSEnabled  bool
	TLSCertFile string
	TLSKeyFile  string
}

// Default configuration values
var defaultConfig = Config{
	ServerName:           "MUD Engine",
	ServerVersion:        "0.1.0",
	ServerHost:           "", // Empty = bind to all interfaces (0.0.0.0)
	ServerPort:           8080,
	DBType:               "sqlite",
	DBHost:               "localhost",
	DBPort:               5432,
	DBName:               "data/mud.db",
	DBUser:               "muduser",
	DBPassword:           "",
	DBMaxConnections:     25,
	DBMaxIdleConns:       5,
	RedisEnabled:         false,
	RedisHost:            "localhost",
	RedisPort:            6379,
	RedisDB:              0,
	MaxPlayers:           100,
	ShutdownTimeoutSecs:  30,
	ReconnectAttempts:    5,
	SessionTimeoutMins:   60,
	TLSEnabled:           false,
	TLSCertFile:          "certs/server.crt",
	TLSKeyFile:           "certs/server.key",
}

// LoadConfig loads configuration from environment file
// Command line flag -env can specify a custom .env file
func LoadConfig() (*Config, error) {
	// Parse command line flags
	envFile := flag.String("env", ".env", "Path to environment configuration file")
	flag.Parse()
	
	log.Printf("Loading configuration from: %s", *envFile)
	
	// Start with default config
	config := defaultConfig
	
	// Try to load from .env file
	if err := loadEnvFile(*envFile, &config); err != nil {
		if os.IsNotExist(err) {
			log.Printf("Configuration file %s not found, creating with defaults...", *envFile)
			if err := createDefaultEnvFile(*envFile); err != nil {
				return nil, fmt.Errorf("failed to create default config: %w", err)
			}
			log.Printf("Created default configuration file: %s", *envFile)
		} else {
			return nil, fmt.Errorf("failed to load config: %w", err)
		}
	}
	
	// Validate configuration
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	
	log.Println("Configuration loaded successfully")
	return &config, nil
}

// loadEnvFile reads configuration from an environment file
func loadEnvFile(filename string, config *Config) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	
	scanner := bufio.NewScanner(file)
	lineNum := 0
	
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		// Parse key=value pairs
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			log.Printf("Warning: Invalid line %d in %s: %s", lineNum, filename, line)
			continue
		}
		
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		
		// Remove quotes if present
		value = strings.Trim(value, "\"'")
		
		// Set configuration value
		if err := setConfigValue(config, key, value); err != nil {
			log.Printf("Warning: Error setting %s on line %d: %v", key, lineNum, err)
		}
	}
	
	return scanner.Err()
}

// setConfigValue sets a configuration value by key name
func setConfigValue(config *Config, key, value string) error {
	switch key {
	// Server settings
	case "SERVER_NAME":
		config.ServerName = value
	case "SERVER_VERSION":
		config.ServerVersion = value
	case "SERVER_HOST":
		config.ServerHost = value
	case "SERVER_PORT":
		port, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		config.ServerPort = port
		
	// Database settings
	case "DB_TYPE":
		config.DBType = value
	case "DB_HOST":
		config.DBHost = value
	case "DB_PORT":
		port, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		config.DBPort = port
	case "DB_NAME":
		config.DBName = value
	case "DB_USER":
		config.DBUser = value
	case "DB_PASSWORD":
		config.DBPassword = value
	case "DB_MAX_CONNECTIONS":
		max, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		config.DBMaxConnections = max
	case "DB_MAX_IDLE_CONNS":
		max, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		config.DBMaxIdleConns = max
		
	// Redis settings
	case "REDIS_ENABLED":
		config.RedisEnabled = value == "true" || value == "1"
	case "REDIS_HOST":
		config.RedisHost = value
	case "REDIS_PORT":
		port, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		config.RedisPort = port
	case "REDIS_DB":
		db, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		config.RedisDB = db
		
	// Server behavior
	case "MAX_PLAYERS":
		max, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		config.MaxPlayers = max
	case "SHUTDOWN_TIMEOUT_SECS":
		timeout, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		config.ShutdownTimeoutSecs = timeout
	case "RECONNECT_ATTEMPTS":
		attempts, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		config.ReconnectAttempts = attempts
	case "SESSION_TIMEOUT_MINS":
		timeout, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		config.SessionTimeoutMins = timeout
		
	// TLS settings
	case "TLS_ENABLED":
		config.TLSEnabled = value == "true" || value == "1"
	case "TLS_CERT_FILE":
		config.TLSCertFile = value
	case "TLS_KEY_FILE":
		config.TLSKeyFile = value
		
	default:
		// Unknown key - just log it
		log.Printf("Warning: Unknown configuration key: %s", key)
	}
	
	return nil
}

// createDefaultEnvFile creates a default .env file with comments
func createDefaultEnvFile(filename string) error {
	content := `# MUD Engine Configuration File
# This file contains bootstrap configuration for the MUD server
# It will be automatically created with defaults if missing

# ==============================================================================
# SERVER SETTINGS
# ==============================================================================
SERVER_NAME=MUD Engine
SERVER_VERSION=0.1.0

# Host/IP to bind to:
#   (empty)      = Bind to all interfaces (0.0.0.0) - accessible from network
#   localhost    = Bind to localhost only (127.0.0.1) - local connections only
#   192.168.1.10 = Bind to specific IP address
SERVER_HOST=

SERVER_PORT=8080

# ==============================================================================
# DATABASE SETTINGS
# ==============================================================================
# DB_TYPE: "sqlite" or "postgres"
DB_TYPE=sqlite

# For SQLite (single file database)
# DB_NAME is the path to the database file
DB_NAME=data/mud.db

# For PostgreSQL (uncomment and configure when migrating)
# DB_HOST=localhost
# DB_PORT=5432
# DB_USER=muduser
# DB_PASSWORD=your_secure_password_here

# Database connection pool settings
DB_MAX_CONNECTIONS=25
DB_MAX_IDLE_CONNS=5

# ==============================================================================
# REDIS SETTINGS (Future Use)
# ==============================================================================
REDIS_ENABLED=false
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_DB=0

# ==============================================================================
# SERVER BEHAVIOR
# ==============================================================================
MAX_PLAYERS=100
SHUTDOWN_TIMEOUT_SECS=30
RECONNECT_ATTEMPTS=5
SESSION_TIMEOUT_MINS=60

# ==============================================================================
# TLS/SSL SETTINGS (Future Use)
# ==============================================================================
TLS_ENABLED=false
TLS_CERT_FILE=certs/server.crt
TLS_KEY_FILE=certs/server.key
`
	
	return os.WriteFile(filename, []byte(content), 0644)
}

// validateConfig checks if configuration values are valid
func validateConfig(config *Config) error {
	if config.ServerPort < 1 || config.ServerPort > 65535 {
		return fmt.Errorf("invalid SERVER_PORT: must be between 1 and 65535")
	}
	
	if config.DBType != "sqlite" && config.DBType != "postgres" {
		return fmt.Errorf("invalid DB_TYPE: must be 'sqlite' or 'postgres'")
	}
	
	if config.DBName == "" {
		return fmt.Errorf("DB_NAME cannot be empty")
	}
	
	if config.DBType == "postgres" {
		if config.DBHost == "" {
			return fmt.Errorf("DB_HOST required for PostgreSQL")
		}
		if config.DBUser == "" {
			return fmt.Errorf("DB_USER required for PostgreSQL")
		}
	}
	
	if config.MaxPlayers < 1 {
		return fmt.Errorf("MAX_PLAYERS must be at least 1")
	}
	
	if config.ShutdownTimeoutSecs < 5 {
		return fmt.Errorf("SHUTDOWN_TIMEOUT_SECS must be at least 5 seconds")
	}
	
	return nil
}

// GetConnectionString returns the database connection string
func (c *Config) GetConnectionString() string {
	switch c.DBType {
	case "sqlite":
		return c.DBName
	case "postgres":
		return fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName,
		)
	default:
		return ""
	}
}

// GetBindAddress returns the address to bind the server to
func (c *Config) GetBindAddress() string {
	if c.ServerHost == "" {
		return "0.0.0.0" // All interfaces
	}
	return c.ServerHost
}

// GetListenAddress returns the full listen address (host:port)
func (c *Config) GetListenAddress() string {
	return fmt.Sprintf("%s:%d", c.GetBindAddress(), c.ServerPort)
}

// LogConfig logs the current configuration (without sensitive data)
func (c *Config) LogConfig() {
	log.Println("=== Server Configuration ===")
	log.Printf("Server: %s v%s", c.ServerName, c.ServerVersion)
	log.Printf("Bind Address: %s:%d", c.GetBindAddress(), c.ServerPort)
	log.Printf("Database Type: %s", c.DBType)
	if c.DBType == "sqlite" {
		log.Printf("Database File: %s", c.DBName)
	} else {
		log.Printf("Database Host: %s:%d", c.DBHost, c.DBPort)
		log.Printf("Database Name: %s", c.DBName)
	}
	log.Printf("Max Players: %d", c.MaxPlayers)
	log.Printf("Redis: %v", c.RedisEnabled)
	log.Printf("TLS: %v", c.TLSEnabled)
	log.Println("===========================")
}

