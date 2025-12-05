package main

import (
	"log"

	"mudengine/internal/config"
	"mudengine/internal/database"
	"mudengine/internal/game"
)

func main() {
	log.Println("=== Room Manager Test ===")
	
	// Load config
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	
	// Initialize database
	if err := database.Initialize(cfg); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()
	
	// Initialize room manager
	if err := game.InitializeRoomManager(); err != nil {
		log.Fatalf("Failed to initialize room manager: %v", err)
	}
	
	log.Printf("Room manager loaded %d rooms", game.Manager.GetRoomCount())
	
	// Test 1: Get a room
	log.Println("\n--- Test 1: Get Builder Room ---")
	room, err := game.Manager.GetRoom("00000000-0000-0000-0000-000000000000")
	if err != nil {
		log.Fatalf("Failed to get room: %v", err)
	}
	log.Printf("Room: %s", room.Title)
	log.Printf("Description: %s", room.Description)
	log.Printf("Exits: %d", len(room.Exits))
	
	// Test 2: Set player location
	log.Println("\n--- Test 2: Set Player Location ---")
	playerID := "test-player-1"
	if err := game.Manager.SetPlayerRoom(playerID, room.ID); err != nil {
		log.Fatalf("Failed to set player room: %v", err)
	}
	log.Printf("Set player %s to room %s", playerID, room.Title)
	
	// Test 3: Get player location
	log.Println("\n--- Test 3: Get Player Location ---")
	playerRoomID, err := game.Manager.GetPlayerRoom(playerID)
	if err != nil {
		log.Fatalf("Failed to get player room: %v", err)
	}
	log.Printf("Player %s is in room: %s", playerID, playerRoomID)
	
	// Test 4: Get players in room
	log.Println("\n--- Test 4: Get Players In Room ---")
	players := game.Manager.GetPlayersInRoom(room.ID)
	log.Printf("Players in %s: %d", room.Title, len(players))
	for _, p := range players {
		log.Printf("  - %s", p)
	}
	
	// Test 5: Add more players
	log.Println("\n--- Test 5: Add Multiple Players ---")
	game.Manager.SetPlayerRoom("player-2", room.ID)
	game.Manager.SetPlayerRoom("player-3", room.ID)
	players = game.Manager.GetPlayersInRoom(room.ID)
	log.Printf("Players in room now: %d", len(players))
	
	// Test 6: Get obvious exits
	log.Println("\n--- Test 6: Get Obvious Exits ---")
	exits, err := game.Manager.GetObviousExits(room.ID)
	if err != nil {
		log.Fatalf("Failed to get exits: %v", err)
	}
	log.Printf("Obvious exits from %s: %d", room.Title, len(exits))
	for _, exit := range exits {
		log.Printf("  - %v -> %s", exit.Keywords, exit.ToRoomID)
	}
	
	// Test 7: Find exit by keyword
	log.Println("\n--- Test 7: Find Exit By Keyword ---")
	if len(exits) > 0 && len(exits[0].Keywords) > 0 {
		keyword := exits[0].Keywords[0]
		foundExit, err := game.Manager.FindExitByKeyword(room.ID, keyword)
		if err != nil {
			log.Printf("No exit found for keyword '%s'", keyword)
		} else {
			log.Printf("Found exit with keyword '%s' leading to room %s", keyword, foundExit.ToRoomID)
		}
	} else {
		log.Println("No exits to test keyword search")
	}
	
	// Test 8: Room stats
	log.Println("\n--- Test 8: Room Statistics ---")
	stats, err := game.Manager.GetRoomStats(room.ID)
	if err != nil {
		log.Fatalf("Failed to get room stats: %v", err)
	}
	log.Printf("Room: %s", stats.Title)
	log.Printf("Players: %d", stats.PlayerCount)
	log.Printf("Exits: %d", stats.ExitCount)
	log.Printf("Darkness: %d", stats.Darkness)
	
	// Test 9: Remove player
	log.Println("\n--- Test 9: Remove Player ---")
	game.Manager.RemovePlayer(playerID)
	players = game.Manager.GetPlayersInRoom(room.ID)
	log.Printf("After removing %s, players in room: %d", playerID, len(players))
	
	// Test 10: Overall stats
	log.Println("\n--- Test 10: Manager Statistics ---")
	log.Printf("Total rooms in cache: %d", game.Manager.GetRoomCount())
	log.Printf("Total tracked players: %d", game.Manager.GetPlayerCount())
	
	log.Println("\n=== All Tests Passed! ===")
}