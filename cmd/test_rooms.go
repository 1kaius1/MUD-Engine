package main

import (
	"fmt"
	"log"

	"mudengine/internal/config"
	"mudengine/internal/database"
)

func main() {
	log.Println("=== Room CRUD Test ===")

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

	// Test 1: Get existing room (Builder Room)
	log.Println("\n--- Test 1: Get Builder Room ---")
	room, err := database.GetRoom("00000000-0000-0000-0000-000000000000")
	if err != nil {
		log.Fatalf("Failed to get builder room: %v", err)
	}
	log.Printf("Found room: %s - %s", room.ID, room.Title)
	log.Printf("Description: %s", room.Description)
	log.Printf("Zone: %s", room.ZoneID)
	log.Printf("Darkness: %d", room.Darkness)

	// Test 2: Create a new zone
	log.Println("\n--- Test 2: Create New Zone ---")
	startingZone := &database.Zone{
		Name:        "Town Square Area",
		Description: "The central gathering place of the town",
		Theme:       "generic",
	}
	if err := database.CreateZone(startingZone); err != nil {
		log.Fatalf("Failed to create zone: %v", err)
	}
	log.Printf("Created zone: %s - %s", startingZone.ID, startingZone.Name)

	// Test 3: Create a new room
	log.Println("\n--- Test 3: Create New Room ---")
	townSquare := &database.Room{
		ZoneID:      startingZone.ID,
		Title:       "The Town Square",
		Description: "You stand in the bustling town square. A large fountain dominates the center, with merchants hawking their wares around its edge. A weathered wooden sign stands near the fountain.",
		Terrain:     "city",
		Darkness:    0,
		Status:      "",
	}
	if err := database.CreateRoom(townSquare); err != nil {
		log.Fatalf("Failed to create room: %v", err)
	}
	log.Printf("Created room: %s - %s", townSquare.ID, townSquare.Title)

	// Test 4: Create another room
	log.Println("\n--- Test 4: Create Second Room ---")
	northSquare := &database.Room{
		ZoneID:      startingZone.ID,
		Title:       "North End of Town Square",
		Description: "The northern section of the town square is quieter, with benches arranged in the shade of large oak trees. A path leads north toward the town gates.",
		Terrain:     "city",
		Darkness:    0,
		Status:      "",
	}
	if err := database.CreateRoom(northSquare); err != nil {
		log.Fatalf("Failed to create room: %v", err)
	}
	log.Printf("Created room: %s - %s", northSquare.ID, northSquare.Title)

	// Test 5: Create an exit between rooms
	log.Println("\n--- Test 5: Create Exit ---")
	exit := &database.Exit{
		FromRoomID:       townSquare.ID,
		ToRoomID:         northSquare.ID,
		Keywords:         []string{"north", "n"},
		Description:      "A cobblestone path leads north",
		IsHidden:         false,
		IsObvious:        true,
		AllowLookThrough: true,
		IsOpen:           true,
		IsLocked:         false,
	}
	if err := database.CreateExit(exit); err != nil {
		log.Fatalf("Failed to create exit: %v", err)
	}
	log.Printf("Created exit: %s from %s to %s", exit.ID, townSquare.Title, northSquare.Title)

	// Create return exit
	returnExit := &database.Exit{
		FromRoomID:       northSquare.ID,
		ToRoomID:         townSquare.ID,
		Keywords:         []string{"south", "s"},
		Description:      "A cobblestone path leads south",
		IsHidden:         false,
		IsObvious:        true,
		AllowLookThrough: true,
		IsOpen:           true,
		IsLocked:         false,
	}
	if err := database.CreateExit(returnExit); err != nil {
		log.Fatalf("Failed to create return exit: %v", err)
	}
	log.Printf("Created return exit: %s", returnExit.ID)

	// Test 6: Retrieve room with exits
	log.Println("\n--- Test 6: Get Room With Exits ---")
	loadedRoom, err := database.GetRoom(townSquare.ID)
	if err != nil {
		log.Fatalf("Failed to load room: %v", err)
	}
	log.Printf("Loaded room: %s", loadedRoom.Title)
	log.Printf("Number of exits: %d", len(loadedRoom.Exits))
	for _, ex := range loadedRoom.Exits {
		log.Printf("  Exit: %v -> Room %s", ex.Keywords, ex.ToRoomID)
	}

	// Test 7: Update a room
	log.Println("\n--- Test 7: Update Room ---")
	townSquare.Description = "You stand in the bustling town square. A large fountain dominates the center, with merchants hawking their wares around its edge. A weathered wooden sign stands near the fountain. The square is more crowded than usual today."
	townSquare.Darkness = 1 // Slightly darker
	if err := database.UpdateRoom(townSquare); err != nil {
		log.Fatalf("Failed to update room: %v", err)
	}
	log.Printf("Updated room: %s", townSquare.Title)

	// Verify update
	updatedRoom, err := database.GetRoom(townSquare.ID)
	if err != nil {
		log.Fatalf("Failed to load updated room: %v", err)
	}
	log.Printf("Darkness is now: %d", updatedRoom.Darkness)

	// Test 8: Get all rooms in zone
	log.Println("\n--- Test 8: Get Rooms By Zone ---")
	rooms, err := database.GetRoomsByZone(startingZone.ID)
	if err != nil {
		log.Fatalf("Failed to get rooms by zone: %v", err)
	}
	log.Printf("Found %d rooms in zone '%s':", len(rooms), startingZone.Name)
	for _, r := range rooms {
		log.Printf("  - %s", r.Title)
	}

	// Test 9: Get all zones
	log.Println("\n--- Test 9: Get All Zones ---")
	zones, err := database.GetAllZones()
	if err != nil {
		log.Fatalf("Failed to get zones: %v", err)
	}
	log.Printf("Found %d zones:", len(zones))
	for _, z := range zones {
		log.Printf("  - %s (%s)", z.Name, z.Theme)
	}

	// Test 10: Delete (optional - uncomment to test deletion)
	// log.Println("\n--- Test 10: Delete Exit ---")
	// if err := database.DeleteExit(exit.ID); err != nil {
	// 	log.Fatalf("Failed to delete exit: %v", err)
	// }
	// log.Printf("Deleted exit: %s", exit.ID)

	log.Println("\n=== All Tests Passed! ===")

	// Print summary
	fmt.Println("\n=== Database Summary ===")
	allRooms, _ := database.GetAllRooms()
	fmt.Printf("Total Rooms: %d\n", len(allRooms))
	fmt.Printf("Total Zones: %d\n", len(zones))
	fmt.Println("========================")
}
