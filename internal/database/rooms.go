package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Room represents a room in the game world
type Room struct {
	ID          string `json:"id"`
	ZoneID      string `json:"zone_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Terrain     string `json:"terrain"`
	Darkness    int    `json:"darkness"`

	// Flags
	BlocksMagic       bool `json:"blocks_magic"`
	RestrictsMovement bool `json:"restricts_movement"`
	NoTeleportIn      bool `json:"no_teleport_in"`
	NoTeleportOut     bool `json:"no_teleport_out"`

	// Traps
	HasTrap          bool `json:"has_trap"`
	TrapDamage       int  `json:"trap_damage"`
	TrapTickInterval int  `json:"trap_tick_interval"`

	// Status effects
	Status string `json:"status"`

	// Metadata
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Runtime data (loaded separately)
	Exits    []*Exit  `json:"exits,omitempty"`
	Objects  []string `json:"objects,omitempty"`  // Object IDs
	Entities []string `json:"entities,omitempty"` // Entity IDs
}

// Exit represents a connection between rooms
type Exit struct {
	ID               string   `json:"id"`
	FromRoomID       string   `json:"from_room_id"`
	ToRoomID         string   `json:"to_room_id"`
	Keywords         []string `json:"keywords"`
	Description      string   `json:"description"`
	IsHidden         bool     `json:"is_hidden"`
	IsObvious        bool     `json:"is_obvious"`
	AllowLookThrough bool     `json:"allow_look_through"`
	IsOpen           bool     `json:"is_open"`
	IsLocked         bool     `json:"is_locked"`
	RequiresItemID   *string  `json:"requires_item_id,omitempty"`
}

// Zone represents a grouping of rooms
type Zone struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Theme       string    `json:"theme"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CreateRoom creates a new room in the database
func CreateRoom(room *Room) error {
	// Generate UUID if not provided
	if room.ID == "" {
		room.ID = uuid.New().String()
	}

	// Set timestamps
	now := time.Now()
	room.CreatedAt = now
	room.UpdatedAt = now

	query := `
		INSERT INTO rooms (
			id, zone_id, title, description, terrain, darkness,
			blocks_magic, restricts_movement, no_teleport_in, no_teleport_out,
			has_trap, trap_damage, trap_tick_interval, status,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := DB.Exec(query,
		room.ID, room.ZoneID, room.Title, room.Description, room.Terrain, room.Darkness,
		room.BlocksMagic, room.RestrictsMovement, room.NoTeleportIn, room.NoTeleportOut,
		room.HasTrap, room.TrapDamage, room.TrapTickInterval, room.Status,
		room.CreatedAt, room.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create room: %w", err)
	}

	return nil
}

// GetRoom retrieves a room by ID
func GetRoom(id string) (*Room, error) {
	room := &Room{}

	query := `
		SELECT 
			id, zone_id, title, description, terrain, darkness,
			blocks_magic, restricts_movement, no_teleport_in, no_teleport_out,
			has_trap, trap_damage, trap_tick_interval, status,
			created_at, updated_at
		FROM rooms
		WHERE id = ?
	`

	err := DB.QueryRow(query, id).Scan(
		&room.ID, &room.ZoneID, &room.Title, &room.Description, &room.Terrain, &room.Darkness,
		&room.BlocksMagic, &room.RestrictsMovement, &room.NoTeleportIn, &room.NoTeleportOut,
		&room.HasTrap, &room.TrapDamage, &room.TrapTickInterval, &room.Status,
		&room.CreatedAt, &room.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("room not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get room: %w", err)
	}

	// Load exits for this room
	exits, err := GetExitsByRoom(id)
	if err != nil {
		return nil, fmt.Errorf("failed to load exits: %w", err)
	}
	room.Exits = exits

	return room, nil
}

// GetRoomsByZone retrieves all rooms in a zone
func GetRoomsByZone(zoneID string) ([]*Room, error) {
	query := `
		SELECT 
			id, zone_id, title, description, terrain, darkness,
			blocks_magic, restricts_movement, no_teleport_in, no_teleport_out,
			has_trap, trap_damage, trap_tick_interval, status,
			created_at, updated_at
		FROM rooms
		WHERE zone_id = ?
		ORDER BY title
	`

	rows, err := DB.Query(query, zoneID)
	if err != nil {
		return nil, fmt.Errorf("failed to query rooms: %w", err)
	}
	defer rows.Close()

	var rooms []*Room
	for rows.Next() {
		room := &Room{}
		err := rows.Scan(
			&room.ID, &room.ZoneID, &room.Title, &room.Description, &room.Terrain, &room.Darkness,
			&room.BlocksMagic, &room.RestrictsMovement, &room.NoTeleportIn, &room.NoTeleportOut,
			&room.HasTrap, &room.TrapDamage, &room.TrapTickInterval, &room.Status,
			&room.CreatedAt, &room.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan room: %w", err)
		}
		rooms = append(rooms, room)
	}

	return rooms, nil
}

// UpdateRoom updates an existing room
func UpdateRoom(room *Room) error {
	room.UpdatedAt = time.Now()

	query := `
		UPDATE rooms SET
			zone_id = ?, title = ?, description = ?, terrain = ?, darkness = ?,
			blocks_magic = ?, restricts_movement = ?, no_teleport_in = ?, no_teleport_out = ?,
			has_trap = ?, trap_damage = ?, trap_tick_interval = ?, status = ?,
			updated_at = ?
		WHERE id = ?
	`

	result, err := DB.Exec(query,
		room.ZoneID, room.Title, room.Description, room.Terrain, room.Darkness,
		room.BlocksMagic, room.RestrictsMovement, room.NoTeleportIn, room.NoTeleportOut,
		room.HasTrap, room.TrapDamage, room.TrapTickInterval, room.Status,
		room.UpdatedAt, room.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update room: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("room not found: %s", room.ID)
	}

	return nil
}

// DeleteRoom deletes a room from the database
func DeleteRoom(id string) error {
	// First delete all exits from/to this room
	_, err := DB.Exec("DELETE FROM exits WHERE from_room_id = ? OR to_room_id = ?", id, id)
	if err != nil {
		return fmt.Errorf("failed to delete room exits: %w", err)
	}

	// Delete the room
	result, err := DB.Exec("DELETE FROM rooms WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete room: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("room not found: %s", id)
	}

	return nil
}

// GetAllRooms retrieves all rooms (use with caution for large databases)
func GetAllRooms() ([]*Room, error) {
	query := `
		SELECT 
			id, zone_id, title, description, terrain, darkness,
			blocks_magic, restricts_movement, no_teleport_in, no_teleport_out,
			has_trap, trap_damage, trap_tick_interval, status,
			created_at, updated_at
		FROM rooms
		ORDER BY title
	`

	rows, err := DB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query rooms: %w", err)
	}
	defer rows.Close()

	var rooms []*Room
	for rows.Next() {
		room := &Room{}
		err := rows.Scan(
			&room.ID, &room.ZoneID, &room.Title, &room.Description, &room.Terrain, &room.Darkness,
			&room.BlocksMagic, &room.RestrictsMovement, &room.NoTeleportIn, &room.NoTeleportOut,
			&room.HasTrap, &room.TrapDamage, &room.TrapTickInterval, &room.Status,
			&room.CreatedAt, &room.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan room: %w", err)
		}
		rooms = append(rooms, room)
	}

	return rooms, nil
}

// CreateExit creates a new exit between rooms
func CreateExit(exit *Exit) error {
	// Generate UUID if not provided
	if exit.ID == "" {
		exit.ID = uuid.New().String()
	}

	// Marshal keywords to JSON
	keywordsJSON, err := json.Marshal(exit.Keywords)
	if err != nil {
		return fmt.Errorf("failed to marshal keywords: %w", err)
	}

	query := `
		INSERT INTO exits (
			id, from_room_id, to_room_id, keywords, description,
			is_hidden, is_obvious, allow_look_through, is_open, is_locked,
			requires_item_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = DB.Exec(query,
		exit.ID, exit.FromRoomID, exit.ToRoomID, string(keywordsJSON), exit.Description,
		exit.IsHidden, exit.IsObvious, exit.AllowLookThrough, exit.IsOpen, exit.IsLocked,
		exit.RequiresItemID,
	)

	if err != nil {
		return fmt.Errorf("failed to create exit: %w", err)
	}

	return nil
}

// GetExitsByRoom retrieves all exits from a room
func GetExitsByRoom(roomID string) ([]*Exit, error) {
	query := `
		SELECT 
			id, from_room_id, to_room_id, keywords, description,
			is_hidden, is_obvious, allow_look_through, is_open, is_locked,
			requires_item_id
		FROM exits
		WHERE from_room_id = ?
	`

	rows, err := DB.Query(query, roomID)
	if err != nil {
		return nil, fmt.Errorf("failed to query exits: %w", err)
	}
	defer rows.Close()

	var exits []*Exit
	for rows.Next() {
		exit := &Exit{}
		var keywordsJSON string
		var requiresItemID sql.NullString

		err := rows.Scan(
			&exit.ID, &exit.FromRoomID, &exit.ToRoomID, &keywordsJSON, &exit.Description,
			&exit.IsHidden, &exit.IsObvious, &exit.AllowLookThrough, &exit.IsOpen, &exit.IsLocked,
			&requiresItemID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan exit: %w", err)
		}

		// Unmarshal keywords
		if err := json.Unmarshal([]byte(keywordsJSON), &exit.Keywords); err != nil {
			return nil, fmt.Errorf("failed to unmarshal keywords: %w", err)
		}

		// Handle nullable requires_item_id
		if requiresItemID.Valid {
			exit.RequiresItemID = &requiresItemID.String
		}

		exits = append(exits, exit)
	}

	return exits, nil
}

// DeleteExit deletes an exit
func DeleteExit(id string) error {
	result, err := DB.Exec("DELETE FROM exits WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete exit: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("exit not found: %s", id)
	}

	return nil
}

// CreateZone creates a new zone
func CreateZone(zone *Zone) error {
	if zone.ID == "" {
		zone.ID = uuid.New().String()
	}

	now := time.Now()
	zone.CreatedAt = now
	zone.UpdatedAt = now

	query := `
		INSERT INTO zones (id, name, description, theme, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err := DB.Exec(query, zone.ID, zone.Name, zone.Description, zone.Theme, zone.CreatedAt, zone.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create zone: %w", err)
	}

	return nil
}

// GetZone retrieves a zone by ID
func GetZone(id string) (*Zone, error) {
	zone := &Zone{}

	query := "SELECT id, name, description, theme, created_at, updated_at FROM zones WHERE id = ?"

	err := DB.QueryRow(query, id).Scan(
		&zone.ID, &zone.Name, &zone.Description, &zone.Theme, &zone.CreatedAt, &zone.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("zone not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get zone: %w", err)
	}

	return zone, nil
}

// GetAllZones retrieves all zones
func GetAllZones() ([]*Zone, error) {
	query := "SELECT id, name, description, theme, created_at, updated_at FROM zones ORDER BY name"

	rows, err := DB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query zones: %w", err)
	}
	defer rows.Close()

	var zones []*Zone
	for rows.Next() {
		zone := &Zone{}
		err := rows.Scan(&zone.ID, &zone.Name, &zone.Description, &zone.Theme, &zone.CreatedAt, &zone.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan zone: %w", err)
		}
		zones = append(zones, zone)
	}

	return zones, nil
}
