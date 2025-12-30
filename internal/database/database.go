// File: internal/database/database.go
// MUD Engine - Database Connection Manager

package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3" // SQLite driver
	"mudengine/internal/config"
)

// DB is the global database connection
var DB *sql.DB

// Initialize opens and initializes the database connection
func Initialize(cfg *config.Config) error {
	log.Println("Initializing database connection...")

	var err error
	
	switch cfg.DBType {
	case "sqlite":
		err = initializeSQLite(cfg)
	case "postgres":
		err = initializePostgreSQL(cfg)
	default:
		return fmt.Errorf("unsupported database type: %s", cfg.DBType)
	}
	
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	
	// Test the connection
	if err := DB.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}
	
	// Set connection pool settings
	DB.SetMaxOpenConns(cfg.DBMaxConnections)
	DB.SetMaxIdleConns(cfg.DBMaxIdleConns)
	
	log.Printf("Database connection established (%s)", cfg.DBType)
	
	// Check if database needs initialization
	needsInit, err := needsInitialization()
	if err != nil {
		return fmt.Errorf("failed to check initialization status: %w", err)
	}
	
	if needsInit {
		log.Println("Database appears to be new, initializing schema...")
		if err := initializeSchema(); err != nil {
			return fmt.Errorf("failed to initialize schema: %w", err)
		}
		log.Println("Database schema initialized successfully")
	} else {
		log.Println("Database schema already exists")
	}
	
	return nil
}

// initializeSQLite sets up SQLite database connection
func initializeSQLite(cfg *config.Config) error {
	// Ensure the data directory exists
	dbDir := filepath.Dir(cfg.DBName)
	if dbDir != "" && dbDir != "." {
		if err := os.MkdirAll(dbDir, 0755); err != nil {
			return fmt.Errorf("failed to create database directory: %w", err)
		}
	}
	
	// Open database connection
	var err error
	DB, err = sql.Open("sqlite3", cfg.DBName)
	if err != nil {
		return fmt.Errorf("failed to open SQLite database: %w", err)
	}
	
	// Enable foreign keys for SQLite
	if _, err := DB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return fmt.Errorf("failed to enable foreign keys: %w", err)
	}
	
	// Set SQLite performance options
	if _, err := DB.Exec("PRAGMA journal_mode = WAL"); err != nil {
		log.Printf("Warning: failed to set WAL mode: %v", err)
	}
	
	return nil
}

// initializePostgreSQL sets up PostgreSQL database connection
func initializePostgreSQL(cfg *config.Config) error {
	// TODO: Implement PostgreSQL connection
	// This is a placeholder for when we migrate to PostgreSQL
	
	connStr := cfg.GetConnectionString()
	var err error
	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to open PostgreSQL database: %w", err)
	}
	
	return nil
}

// needsInitialization checks if the database schema needs to be created
func needsInitialization() (bool, error) {
	// Check if the zones table exists
	var tableName string
	query := `
		SELECT name FROM sqlite_master 
		WHERE type='table' AND name='zones'
	`
	
	err := DB.QueryRow(query).Scan(&tableName)
	if err == sql.ErrNoRows {
		// Table doesn't exist, needs initialization
		return true, nil
	}
	if err != nil {
		return false, err
	}
	
	// Table exists
	return false, nil
}

// initializeSchema creates all database tables
func initializeSchema() error {
	// Read and execute the schema SQL
	// For now, we'll define it inline. Later we can move to a separate file.
	
	schema := `
-- Zones/Areas/Districts
CREATE TABLE IF NOT EXISTS zones (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    theme TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Rooms
CREATE TABLE IF NOT EXISTS rooms (
    id TEXT PRIMARY KEY,
    zone_id TEXT NOT NULL,
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    terrain TEXT DEFAULT 'indoor',
    darkness INTEGER DEFAULT 0,
    blocks_magic BOOLEAN DEFAULT 0,
    restricts_movement BOOLEAN DEFAULT 0,
    no_teleport_in BOOLEAN DEFAULT 0,
    no_teleport_out BOOLEAN DEFAULT 0,
    has_trap BOOLEAN DEFAULT 0,
    trap_damage INTEGER DEFAULT 0,
    trap_tick_interval INTEGER DEFAULT 0,
    status TEXT DEFAULT '',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (zone_id) REFERENCES zones(id)
);

-- Exits
CREATE TABLE IF NOT EXISTS exits (
    id TEXT PRIMARY KEY,
    from_room_id TEXT NOT NULL,
    to_room_id TEXT NOT NULL,
    keywords TEXT NOT NULL,
    description TEXT,
    is_hidden BOOLEAN DEFAULT 0,
    is_obvious BOOLEAN DEFAULT 1,
    allow_look_through BOOLEAN DEFAULT 1,
    is_open BOOLEAN DEFAULT 1,
    is_locked BOOLEAN DEFAULT 0,
    requires_item_id TEXT,
    FOREIGN KEY (from_room_id) REFERENCES rooms(id),
    FOREIGN KEY (to_room_id) REFERENCES rooms(id),
    FOREIGN KEY (requires_item_id) REFERENCES game_objects(id)
);

-- Game Objects
CREATE TABLE IF NOT EXISTS game_objects (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL,
    container_id TEXT,
    container_type TEXT,
    object_type TEXT NOT NULL,
    is_obvious BOOLEAN DEFAULT 1,
    is_hidden BOOLEAN DEFAULT 0,
    can_pick_up BOOLEAN DEFAULT 1,
    is_readable BOOLEAN DEFAULT 0,
    read_text TEXT,
    is_container BOOLEAN DEFAULT 0,
    capacity REAL DEFAULT 0.0,
    is_open BOOLEAN DEFAULT 1,
    weight REAL DEFAULT 0.0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Entities
CREATE TABLE IF NOT EXISTS entities (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL,
    room_id TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    darkvision INTEGER DEFAULT 0,
    is_hidden BOOLEAN DEFAULT 0,
    health INTEGER DEFAULT 100,
    max_health INTEGER DEFAULT 100,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (room_id) REFERENCES rooms(id)
);

-- Players
CREATE TABLE IF NOT EXISTS players (
    id TEXT PRIMARY KEY,
    entity_id TEXT NOT NULL UNIQUE,
    username TEXT UNIQUE NOT NULL,
    password_hash TEXT,
    mfa_secret TEXT,
    last_login TIMESTAMP,
    last_logout TIMESTAMP,
    is_builder BOOLEAN DEFAULT 0,
    is_admin BOOLEAN DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (entity_id) REFERENCES entities(id)
);

-- NPCs
CREATE TABLE IF NOT EXISTS npcs (
    id TEXT PRIMARY KEY,
    entity_id TEXT NOT NULL UNIQUE,
    is_aggressive BOOLEAN DEFAULT 0,
    is_merchant BOOLEAN DEFAULT 0,
    greeting TEXT,
    FOREIGN KEY (entity_id) REFERENCES entities(id)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_objects_container ON game_objects(container_id);
CREATE INDEX IF NOT EXISTS idx_objects_container_type ON game_objects(container_type);
CREATE INDEX IF NOT EXISTS idx_exits_from_room ON exits(from_room_id);
CREATE INDEX IF NOT EXISTS idx_exits_keywords ON exits(keywords);
CREATE INDEX IF NOT EXISTS idx_rooms_zone ON rooms(zone_id);
CREATE INDEX IF NOT EXISTS idx_entities_room ON entities(room_id);
CREATE INDEX IF NOT EXISTS idx_players_username ON players(username);
`

	// Execute the schema
	if _, err := DB.Exec(schema); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}
	
	log.Println("Database tables created successfully")
	
	// Insert initial data
	if err := insertInitialData(); err != nil {
		return fmt.Errorf("failed to insert initial data: %w", err)
	}
	
	return nil
}

// insertInitialData adds the default zones and rooms
func insertInitialData() error {
	log.Println("Inserting initial data...")
	
	// Insert Staff Area zone
	_, err := DB.Exec(`
		INSERT INTO zones (id, name, description, theme) 
		VALUES (?, ?, ?, ?)
	`, "00000000-0000-0000-0000-000000000001", "Staff Area", "Administrative and building zone", "meta")
	if err != nil {
		return fmt.Errorf("failed to insert staff zone: %w", err)
	}
	
	// Insert Builder Room (Room 0)
	_, err = DB.Exec(`
		INSERT INTO rooms (id, zone_id, title, description, darkness, status)
		VALUES (?, ?, ?, ?, ?, ?)
	`,
		"00000000-0000-0000-0000-000000000000",
		"00000000-0000-0000-0000-000000000001",
		"The Builder Break Room",
		"A comfortable room filled with workbenches, blueprints, and half-finished creations. A coffee pot sits perpetually full in the corner. This is a safe space for staff to chat and work on building the world.",
		0,
		"")
	if err != nil {
		return fmt.Errorf("failed to insert builder room: %w", err)
	}
	
	// Insert Starting Area zone
	_, err = DB.Exec(`
		INSERT INTO zones (id, name, description, theme)
		VALUES (?, ?, ?, ?)
	`, "10000000-0000-0000-0000-000000000001", "Starting Area", "Where new players begin their journey", "generic")
	if err != nil {
		return fmt.Errorf("failed to insert starting zone: %w", err)
	}
	
	log.Println("Initial data inserted successfully")
	return nil
}

// Close closes the database connection
func Close() error {
	if DB != nil {
		log.Println("Closing database connection...")
		return DB.Close()
	}
	return nil
}