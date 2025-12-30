// File: internal/game/room_manager.go
// MUD Engine - Room Management System

package game

import (
	"fmt"
	"log"
	"sync"

	"mudengine/internal/database"
)

// RoomManager manages all rooms in memory
type RoomManager struct {
	rooms       map[string]*database.Room // roomID -> Room
	playerRooms map[string]string         // playerID -> roomID
	mu          sync.RWMutex
}

// Global room manager instance
var Manager *RoomManager

// InitializeRoomManager creates and initializes the room manager
func InitializeRoomManager() error {
	log.Println("Initializing room manager...")
	
	Manager = &RoomManager{
		rooms:       make(map[string]*database.Room),
		playerRooms: make(map[string]string),
	}
	
	// Load all rooms into memory
	if err := Manager.LoadAllRooms(); err != nil {
		return fmt.Errorf("failed to load rooms: %w", err)
	}
	
	log.Printf("Room manager initialized with %d rooms", len(Manager.rooms))
	return nil
}

// LoadAllRooms loads all rooms from the database into memory
func (rm *RoomManager) LoadAllRooms() error {
	rooms, err := database.GetAllRooms()
	if err != nil {
		return err
	}
	
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	for _, room := range rooms {
		// Load exits for each room
		exits, err := database.GetExitsByRoom(room.ID)
		if err != nil {
			log.Printf("Warning: failed to load exits for room %s: %v", room.ID, err)
			continue
		}
		room.Exits = exits
		
		rm.rooms[room.ID] = room
	}
	
	return nil
}

// LoadRoom loads a single room from database into cache
func (rm *RoomManager) LoadRoom(roomID string) (*database.Room, error) {
	// Check if already in cache
	rm.mu.RLock()
	if room, exists := rm.rooms[roomID]; exists {
		rm.mu.RUnlock()
		return room, nil
	}
	rm.mu.RUnlock()
	
	// Load from database
	room, err := database.GetRoom(roomID)
	if err != nil {
		return nil, err
	}
	
	// Cache it
	rm.mu.Lock()
	rm.rooms[roomID] = room
	rm.mu.Unlock()
	
	return room, nil
}

// GetRoom retrieves a room from cache (or loads it)
func (rm *RoomManager) GetRoom(roomID string) (*database.Room, error) {
	rm.mu.RLock()
	room, exists := rm.rooms[roomID]
	rm.mu.RUnlock()
	
	if !exists {
		return rm.LoadRoom(roomID)
	}
	
	return room, nil
}

// GetPlayerRoom returns the room ID where a player is located
func (rm *RoomManager) GetPlayerRoom(playerID string) (string, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	
	roomID, exists := rm.playerRooms[playerID]
	if !exists {
		return "", fmt.Errorf("player location not set: %s", playerID)
	}
	
	return roomID, nil
}

// SetPlayerRoom sets the player's current room
func (rm *RoomManager) SetPlayerRoom(playerID, roomID string) error {
	// Verify room exists
	if _, err := rm.GetRoom(roomID); err != nil {
		return fmt.Errorf("room does not exist: %s", roomID)
	}
	
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	rm.playerRooms[playerID] = roomID
	return nil
}

// MovePlayer moves a player from one room to another
func (rm *RoomManager) MovePlayer(playerID, fromRoomID, toRoomID string) error {
	// Verify destination room exists
	if _, err := rm.GetRoom(toRoomID); err != nil {
		return fmt.Errorf("destination room does not exist: %s", toRoomID)
	}
	
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	// Update player location
	rm.playerRooms[playerID] = toRoomID
	
	return nil
}

// GetPlayersInRoom returns all player IDs in a given room
func (rm *RoomManager) GetPlayersInRoom(roomID string) []string {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	
	var players []string
	for playerID, playerRoomID := range rm.playerRooms {
		if playerRoomID == roomID {
			players = append(players, playerID)
		}
	}
	
	return players
}

// RemovePlayer removes a player from tracking (on disconnect)
func (rm *RoomManager) RemovePlayer(playerID string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	delete(rm.playerRooms, playerID)
}

// FindExitByKeyword finds an exit in a room by keyword
func (rm *RoomManager) FindExitByKeyword(roomID, keyword string) (*database.Exit, error) {
	room, err := rm.GetRoom(roomID)
	if err != nil {
		return nil, err
	}
	
	for _, exit := range room.Exits {
		for _, kw := range exit.Keywords {
			if kw == keyword {
				return exit, nil
			}
		}
	}
	
	return nil, fmt.Errorf("no exit found with keyword: %s", keyword)
}

// GetObviousExits returns all non-hidden exits from a room
func (rm *RoomManager) GetObviousExits(roomID string) ([]*database.Exit, error) {
	room, err := rm.GetRoom(roomID)
	if err != nil {
		return nil, err
	}
	
	var obvious []*database.Exit
	for _, exit := range room.Exits {
		if !exit.IsHidden && exit.IsObvious {
			obvious = append(obvious, exit)
		}
	}
	
	return obvious, nil
}

// GetAllExits returns all exits from a room (including hidden)
func (rm *RoomManager) GetAllExits(roomID string) ([]*database.Exit, error) {
	room, err := rm.GetRoom(roomID)
	if err != nil {
		return nil, err
	}
	
	return room.Exits, nil
}

// ReloadRoom refreshes a room from the database
// Useful after builder edits
func (rm *RoomManager) ReloadRoom(roomID string) error {
	room, err := database.GetRoom(roomID)
	if err != nil {
		return err
	}
	
	rm.mu.Lock()
	rm.rooms[roomID] = room
	rm.mu.Unlock()
	
	log.Printf("Reloaded room: %s", roomID)
	return nil
}

// CreateAndCacheRoom creates a new room and adds it to cache
func (rm *RoomManager) CreateAndCacheRoom(room *database.Room) error {
	// Save to database
	if err := database.CreateRoom(room); err != nil {
		return err
	}
	
	// Add to cache
	rm.mu.Lock()
	rm.rooms[room.ID] = room
	rm.mu.Unlock()
	
	log.Printf("Created and cached room: %s", room.Title)
	return nil
}

// GetRoomCount returns the total number of rooms in cache
func (rm *RoomManager) GetRoomCount() int {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	
	return len(rm.rooms)
}

// GetPlayerCount returns the number of tracked players
func (rm *RoomManager) GetPlayerCount() int {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	
	return len(rm.playerRooms)
}

// GetRoomStats returns statistics about a room
type RoomStats struct {
	RoomID       string
	Title        string
	PlayerCount  int
	ExitCount    int
	Darkness     int
}

func (rm *RoomManager) GetRoomStats(roomID string) (*RoomStats, error) {
	room, err := rm.GetRoom(roomID)
	if err != nil {
		return nil, err
	}
	
	stats := &RoomStats{
		RoomID:      room.ID,
		Title:       room.Title,
		PlayerCount: len(rm.GetPlayersInRoom(roomID)),
		ExitCount:   len(room.Exits),
		Darkness:    room.Darkness,
	}
	
	return stats, nil
}