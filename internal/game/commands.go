// File: internal/game/commands.go
// MUD Engine - Command System
//
// Lock/Key Authorization System:
// Commands have "locks" that require specific "keys" to access.
// Players possess "keys" which are granted based on their role/permissions.
// 
// Standard Keys:
//   admin       - Full administrative access
//   builder     - Can create/edit rooms, exits, zones
//   moderator   - Can kick/ban players, moderate chat
//   storyteller - Can run events, control NPCs, spawn items
//
// Usage in commands:
//   if !player.HasKey("builder") { return "Permission denied" }
//   if player.HasAllKeys("admin", "builder") { ... }
//   if player.HasAnyKey("moderator", "admin") { ... }

package game

import (
	"fmt"
	"log"
	"strings"

	"mudengine/internal/database"
)

// CommandHandler is a function that processes a command
type CommandHandler func(player *Player, args []string) string

// Player represents a player for command processing
type Player struct {
	ID            string
	Username      string
	CurrentRoomID string
	Keys          map[string]bool // Keys the player possesses (keyAdmin, keyBuilder, etc.)
}

// HasKey checks if the player possesses a specific key
func (p *Player) HasKey(keyName string) bool {
	if p.Keys == nil {
		return false
	}
	return p.Keys[keyName]
}

// HasAllKeys checks if the player possesses all specified keys
func (p *Player) HasAllKeys(keyNames ...string) bool {
	for _, key := range keyNames {
		if !p.HasKey(key) {
			return false
		}
	}
	return true
}

// HasAnyKey checks if the player possesses at least one of the specified keys
func (p *Player) HasAnyKey(keyNames ...string) bool {
	for _, key := range keyNames {
		if p.HasKey(key) {
			return true
		}
	}
	return false
}

// CommandRegistry holds all available commands
type CommandRegistry struct {
	commands map[string]CommandHandler
}

// Global command registry
var Registry *CommandRegistry

// InitializeCommands sets up the global command registry
// This must be called once at server startup
func InitializeCommands() {
	Registry = NewCommandRegistry()
	log.Printf("Command registry initialized with %d commands", len(Registry.commands))
}

// NewCommandRegistry creates a new command registry
func NewCommandRegistry() *CommandRegistry {
	registry := &CommandRegistry{
		commands: make(map[string]CommandHandler),
	}
	
	// Register standard commands
	registry.Register("look", CmdLook)
	registry.Register("l", CmdLook)
	registry.Register("move", CmdMove)
	registry.Register("quit", CmdQuit)
	registry.Register("say", CmdSay)
	
	// Register directional shortcuts (all call CmdMove with the direction)
	registry.Register("north", func(p *Player, args []string) string { return CmdMove(p, []string{"north"}) })
	registry.Register("n", func(p *Player, args []string) string { return CmdMove(p, []string{"north"}) })
	registry.Register("south", func(p *Player, args []string) string { return CmdMove(p, []string{"south"}) })
	registry.Register("s", func(p *Player, args []string) string { return CmdMove(p, []string{"south"}) })
	registry.Register("east", func(p *Player, args []string) string { return CmdMove(p, []string{"east"}) })
	registry.Register("e", func(p *Player, args []string) string { return CmdMove(p, []string{"east"}) })
	registry.Register("west", func(p *Player, args []string) string { return CmdMove(p, []string{"west"}) })
	registry.Register("w", func(p *Player, args []string) string { return CmdMove(p, []string{"west"}) })
	registry.Register("northeast", func(p *Player, args []string) string { return CmdMove(p, []string{"northeast"}) })
	registry.Register("ne", func(p *Player, args []string) string { return CmdMove(p, []string{"northeast"}) })
	registry.Register("northwest", func(p *Player, args []string) string { return CmdMove(p, []string{"northwest"}) })
	registry.Register("nw", func(p *Player, args []string) string { return CmdMove(p, []string{"northwest"}) })
	registry.Register("southeast", func(p *Player, args []string) string { return CmdMove(p, []string{"southeast"}) })
	registry.Register("se", func(p *Player, args []string) string { return CmdMove(p, []string{"southeast"}) })
	registry.Register("southwest", func(p *Player, args []string) string { return CmdMove(p, []string{"southwest"}) })
	registry.Register("sw", func(p *Player, args []string) string { return CmdMove(p, []string{"southwest"}) })
	registry.Register("up", func(p *Player, args []string) string { return CmdMove(p, []string{"up"}) })
	registry.Register("u", func(p *Player, args []string) string { return CmdMove(p, []string{"up"}) })
	registry.Register("down", func(p *Player, args []string) string { return CmdMove(p, []string{"down"}) })
	registry.Register("d", func(p *Player, args []string) string { return CmdMove(p, []string{"down"}) })
	
	// Register builder/admin commands (no prefix - they're regular game commands)
	registry.Register("teleport", CmdTeleport)
	registry.Register("tp", CmdTeleport)
	registry.Register("goto", CmdTeleport)
	registry.Register("rooms", CmdListRooms)
	registry.Register("zones", CmdListZones)
	registry.Register("room", CmdRoom)
	registry.Register("exit", CmdExit)
	registry.Register("zone", CmdZone)
	
	return registry
}

// Register adds a command to the registry
func (cr *CommandRegistry) Register(name string, handler CommandHandler) {
	cr.commands[strings.ToLower(name)] = handler
}

// Execute runs a command
func (cr *CommandRegistry) Execute(player *Player, input string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return ""
	}
	
	parts := strings.Fields(input)
	cmdName := strings.ToLower(parts[0])
	args := parts[1:]
	
	log.Printf("Executing command: '%s' with args: %v", cmdName, args)
	
	handler, exists := cr.commands[cmdName]
	if !exists {
		log.Printf("Command not found: '%s'", cmdName)
		return fmt.Sprintf("Unknown command: %s\r\n", cmdName)
	}
	
	return handler(player, args)
}

// CmdLook shows the current room description
func CmdLook(player *Player, args []string) string {
	room, err := Manager.GetRoom(player.CurrentRoomID)
	if err != nil {
		return fmt.Sprintf("Error: Unable to look around. %v\r\n", err)
	}
	
	return FormatRoomDescription(room)
}

// CmdMove moves the player in a direction
func CmdMove(player *Player, args []string) string {
	if len(args) == 0 {
		return "Move where?\r\nDirections: north, south, east, west, northeast, northwest, southeast, southwest, up, down\r\nType 'help move' for more information.\r\n"
	}
	
	direction := strings.ToLower(args[0])
	return MovePlayer(player, direction)
}

// MovePlayer handles player movement
func MovePlayer(player *Player, direction string) string {
	// Find exit by keyword
	exit, err := Manager.FindExitByKeyword(player.CurrentRoomID, direction)
	if err != nil {
		return fmt.Sprintf("You can't go %s.\r\n", direction)
	}
	
	// Check if exit is locked
	if exit.IsLocked {
		return "That way is locked.\r\n"
	}
	
	// Check if exit is open
	if !exit.IsOpen {
		return "That way is closed.\r\n"
	}
	
	// TODO: Check if player has required item (key)
	if exit.RequiresItemID != nil {
		return "You need a key to go that way.\r\n"
	}
	
	// Move the player
	oldRoomID := player.CurrentRoomID
	newRoomID := exit.ToRoomID
	
	if err := Manager.MovePlayer(player.ID, oldRoomID, newRoomID); err != nil {
		return fmt.Sprintf("Error moving: %v\r\n", err)
	}
	
	// Update player's current room
	player.CurrentRoomID = newRoomID
	
	// Get new room
	newRoom, err := Manager.GetRoom(newRoomID)
	if err != nil {
		return fmt.Sprintf("Error loading new room: %v\r\n", err)
	}
	
	// TODO: Broadcast to old room: "PlayerName leaves north."
	// TODO: Broadcast to new room: "PlayerName arrives from the south."
	
	// Return the new room description
	return FormatRoomDescription(newRoom)
}

// CmdQuit handles the quit command
func CmdQuit(player *Player, args []string) string {
	return "QUIT" // Special return value to signal disconnect
}

// CmdSay handles the say command
func CmdSay(player *Player, args []string) string {
	if len(args) == 0 {
		return "Say what?\r\n"
	}
	
	message := strings.Join(args, " ")
	
	// TODO: Broadcast to all players in room
	return fmt.Sprintf("You say, \"%s\"\r\n", message)
}

// CmdTeleport teleports a builder/admin to a room
func CmdTeleport(player *Player, args []string) string {
	// Check permissions - requires keyBuilder
	if !player.HasKey("keyBuilder") {
		return "You don't have permission to use this command.\r\n"
	}
	
	if len(args) == 0 {
		return "Usage: teleport <room_id_or_title>\r\n"
	}
	
	target := strings.Join(args, " ")
	
	// Try to find room by ID first (UUID)
	room, err := Manager.GetRoom(target)
	if err != nil {
		// Not found by ID, try to find by title
		room, err = FindRoomByTitle(target)
		if err != nil {
			return fmt.Sprintf("Room not found: %s\r\n", target)
		}
	}
	
	// Move the player
	oldRoomID := player.CurrentRoomID
	if err := Manager.MovePlayer(player.ID, oldRoomID, room.ID); err != nil {
		return fmt.Sprintf("Error teleporting: %v\r\n", err)
	}
	
	// Update player's current room
	player.CurrentRoomID = room.ID
	
	// Return the new room description
	result := fmt.Sprintf("You teleport to %s.\r\n\r\n", room.Title)
	result += FormatRoomDescription(room)
	return result
}

// CmdListRooms lists all rooms (builder command)
func CmdListRooms(player *Player, args []string) string {
	if !player.HasKey("keyBuilder") {
		return "You don't have permission to use this command.\r\n"
	}
	
	rooms, err := database.GetAllRooms()
	if err != nil {
		return fmt.Sprintf("Error listing rooms: %v\r\n", err)
	}
	
	result := "╔════════════════════════════════════════════════════════════════╗\r\n"
	result += "║                         ROOM LIST                              ║\r\n"
	result += "╚════════════════════════════════════════════════════════════════╝\r\n\r\n"
	
	// Group by zone
	zoneRooms := make(map[string][]*database.Room)
	zones := make(map[string]*database.Zone)
	
	for _, room := range rooms {
		zoneRooms[room.ZoneID] = append(zoneRooms[room.ZoneID], room)
		if _, exists := zones[room.ZoneID]; !exists {
			zone, err := database.GetZone(room.ZoneID)
			if err == nil {
				zones[room.ZoneID] = zone
			}
		}
	}
	
	// Display rooms by zone
	for zoneID, roomList := range zoneRooms {
		zone := zones[zoneID]
		if zone != nil {
			result += fmt.Sprintf("Zone: %s\r\n", zone.Name)
			result += strings.Repeat("-", 60) + "\r\n"
		}
		
		for _, room := range roomList {
			result += fmt.Sprintf("  %s\r\n", room.Title)
			result += fmt.Sprintf("    ID: %s\r\n", room.ID)
			result += fmt.Sprintf("    Exits: %d  Darkness: %d\r\n", len(room.Exits), room.Darkness)
		}
		result += "\r\n"
	}
	
	result += fmt.Sprintf("Total: %d rooms\r\n", len(rooms))
	return result
}

// CmdListZones lists all zones (builder command)
func CmdListZones(player *Player, args []string) string {
	if !player.HasKey("keyBuilder") {
		return "You don't have permission to use this command.\r\n"
	}
	
	zones, err := database.GetAllZones()
	if err != nil {
		return fmt.Sprintf("Error listing zones: %v\r\n", err)
	}
	
	result := "╔════════════════════════════════════════════════════════════════╗\r\n"
	result += "║                         ZONE LIST                              ║\r\n"
	result += "╚════════════════════════════════════════════════════════════════╝\r\n\r\n"
	
	for _, zone := range zones {
		result += fmt.Sprintf("%s (%s)\r\n", zone.Name, zone.Theme)
		result += fmt.Sprintf("  ID: %s\r\n", zone.ID)
		result += fmt.Sprintf("  %s\r\n", zone.Description)
		
		// Count rooms in this zone
		rooms, _ := database.GetRoomsByZone(zone.ID)
		result += fmt.Sprintf("  Rooms: %d\r\n\r\n", len(rooms))
	}
	
	result += fmt.Sprintf("Total: %d zones\r\n", len(zones))
	return result
}

// FindRoomByTitle finds a room by its title (case-insensitive partial match)
func FindRoomByTitle(title string) (*database.Room, error) {
	rooms, err := database.GetAllRooms()
	if err != nil {
		return nil, err
	}
	
	titleLower := strings.ToLower(title)
	
	// First try exact match
	for _, room := range rooms {
		if strings.ToLower(room.Title) == titleLower {
			return room, nil
		}
	}
	
	// Then try partial match
	for _, room := range rooms {
		if strings.Contains(strings.ToLower(room.Title), titleLower) {
			return room, nil
		}
	}
	
	return nil, fmt.Errorf("no room found matching: %s", title)
}

// FormatRoomDescription formats a room description for display
func FormatRoomDescription(room *database.Room) string {
	result := fmt.Sprintf("%s\r\n", room.Title)
	result += fmt.Sprintf("%s\r\n\r\n", room.Description)
	
	// Get obvious exits
	obviousExits, err := Manager.GetObviousExits(room.ID)
	if err == nil && len(obviousExits) > 0 {
		exitNames := make([]string, 0, len(obviousExits))
		for _, exit := range obviousExits {
			if len(exit.Keywords) > 0 {
				exitNames = append(exitNames, exit.Keywords[0])
			}
		}
		
		if len(exitNames) > 0 {
			result += fmt.Sprintf("Obvious exits: %s\r\n", strings.Join(exitNames, ", "))
		}
	} else {
		result += "There are no obvious exits.\r\n"
	}
	
	// TODO: Show objects in room
	// TODO: Show other players/NPCs in room
	
	result += "\r\n"
	return result
}

// CmdRoom handles room building commands
func CmdRoom(player *Player, args []string) string {
	if !player.HasKey("keyBuilder") {
		return "You don't have permission to use this command.\r\n"
	}
	
	if len(args) == 0 {
		return "Room commands:\r\n" +
			"  room create <title>     - Create a new room here\r\n" +
			"  room edit <field>       - Edit current room\r\n" +
			"  room info               - Show current room details\r\n" +
			"  room delete <room_id>   - Delete a room (use with caution)\r\n"
	}
	
	subCmd := strings.ToLower(args[0])
	subArgs := args[1:]
	
	switch subCmd {
	case "create":
		return CmdRoomCreate(player, subArgs)
	case "edit":
		return CmdRoomEdit(player, subArgs)
	case "info":
		return CmdRoomInfo(player, subArgs)
	case "delete":
		return CmdRoomDelete(player, subArgs)
	default:
		return fmt.Sprintf("Unknown room command: %s\r\n", subCmd)
	}
}

// CmdRoomCreate creates a new room
func CmdRoomCreate(player *Player, args []string) string {
	if len(args) == 0 {
		return "Usage: room create <title>\r\nExample: room create The Dark Forest\r\n"
	}
	
	title := strings.Join(args, " ")
	
	// Get current room to inherit zone
	currentRoom, err := Manager.GetRoom(player.CurrentRoomID)
	if err != nil {
		return fmt.Sprintf("Error: Cannot determine current location: %v\r\n", err)
	}
	
	// Create new room
	newRoom := &database.Room{
		ZoneID:      currentRoom.ZoneID,
		Title:       title,
		Description: "A newly created room. Use 'room edit description' to set the description.",
		Terrain:     "indoor",
		Darkness:    0,
	}
	
	if err := database.CreateRoom(newRoom); err != nil {
		return fmt.Sprintf("Error creating room: %v\r\n", err)
	}
	
	// Add to room manager cache
	Manager.LoadRoom(newRoom.ID)
	
	return fmt.Sprintf("Created room: %s\r\nRoom ID: %s\r\nUse 'exit create <direction> %s' to create an exit to this room.\r\n",
		newRoom.Title, newRoom.ID, newRoom.ID)
}

// CmdRoomEdit edits the current room
func CmdRoomEdit(player *Player, args []string) string {
	if len(args) == 0 {
		return "Usage: room edit <field> <value>\r\n" +
			"Fields: title, description, terrain, darkness\r\n" +
			"Example: room edit description A dark and foreboding forest path.\r\n"
	}
	
	field := strings.ToLower(args[0])
	if len(args) < 2 {
		return fmt.Sprintf("Please provide a value for %s\r\n", field)
	}
	value := strings.Join(args[1:], " ")
	
	// Get current room
	room, err := Manager.GetRoom(player.CurrentRoomID)
	if err != nil {
		return fmt.Sprintf("Error loading room: %v\r\n", err)
	}
	
	// Update field
	switch field {
	case "title":
		room.Title = value
	case "description", "desc":
		room.Description = value
	case "terrain":
		room.Terrain = value
	case "darkness":
		darkness := 0
		fmt.Sscanf(value, "%d", &darkness)
		if darkness < 0 || darkness > 10 {
			return "Darkness must be between 0 (daylight) and 10 (absolute darkness).\r\n"
		}
		room.Darkness = darkness
	default:
		return fmt.Sprintf("Unknown field: %s\r\n", field)
	}
	
	// Save to database
	if err := database.UpdateRoom(room); err != nil {
		return fmt.Sprintf("Error updating room: %v\r\n", err)
	}
	
	// Reload in cache
	Manager.ReloadRoom(room.ID)
	
	return fmt.Sprintf("Updated %s.\r\n", field)
}

// CmdRoomInfo shows current room info
func CmdRoomInfo(player *Player, args []string) string {
	room, err := Manager.GetRoom(player.CurrentRoomID)
	if err != nil {
		return fmt.Sprintf("Error loading room: %v\r\n", err)
	}
	
	result := "╔════════════════════════════════════════════════════════════════╗\r\n"
	result += "║                       ROOM INFORMATION                         ║\r\n"
	result += "╚════════════════════════════════════════════════════════════════╝\r\n\r\n"
	
	result += fmt.Sprintf("ID:          %s\r\n", room.ID)
	result += fmt.Sprintf("Title:       %s\r\n", room.Title)
	result += fmt.Sprintf("Zone:        %s\r\n", room.ZoneID)
	result += fmt.Sprintf("Terrain:     %s\r\n", room.Terrain)
	result += fmt.Sprintf("Darkness:    %d/10\r\n", room.Darkness)
	result += fmt.Sprintf("Description: %s\r\n\r\n", room.Description)
	
	result += "Exits:\r\n"
	if len(room.Exits) > 0 {
		for _, exit := range room.Exits {
			keywords := strings.Join(exit.Keywords, ", ")
			result += fmt.Sprintf("  [%s] -> %s", keywords, exit.ToRoomID)
			if exit.IsHidden {
				result += " (hidden)"
			}
			if exit.IsLocked {
				result += " (locked)"
			}
			result += "\r\n"
		}
	} else {
		result += "  None\r\n"
	}
	
	return result
}

// CmdRoomDelete deletes a room
func CmdRoomDelete(player *Player, args []string) string {
	if len(args) == 0 {
		return "Usage: room delete <room_id>\r\nWarning: This permanently deletes the room!\r\n"
	}
	
	roomID := args[0]
	
	// Basic validation - check if players are in room
	players := Manager.GetPlayersInRoom(roomID)
	if len(players) > 0 {
		return fmt.Sprintf("Cannot delete room: %d player(s) currently in room.\r\n", len(players))
	}
	
	// Delete from database
	if err := database.DeleteRoom(roomID); err != nil {
		return fmt.Sprintf("Error deleting room: %v\r\n", err)
	}
	
	// Reload room manager
	Manager.LoadAllRooms()
	
	return "Room deleted successfully.\r\n"
}

// CmdExit handles exit building commands
func CmdExit(player *Player, args []string) string {
	if !player.HasKey("keyBuilder") {
		return "You don't have permission to use this command.\r\n"
	}
	
	if len(args) == 0 {
		return "Exit commands:\r\n" +
			"  exit create <direction> <room_id>  - Create an exit\r\n" +
			"  exit delete <direction>             - Delete an exit\r\n" +
			"  exit list                           - List all exits\r\n"
	}
	
	subCmd := strings.ToLower(args[0])
	subArgs := args[1:]
	
	switch subCmd {
	case "create":
		return CmdExitCreate(player, subArgs)
	case "delete":
		return CmdExitDelete(player, subArgs)
	case "list":
		return CmdExitList(player, subArgs)
	default:
		return fmt.Sprintf("Unknown exit command: %s\r\n", subCmd)
	}
}

// CmdExitCreate creates a new exit
func CmdExitCreate(player *Player, args []string) string {
	if len(args) < 2 {
		return "Usage: exit create <direction> <destination_room_id>\r\n" +
			"Example: exit create north abc-123-def\r\n" +
			"Shortcuts: n, s, e, w, ne, nw, se, sw, u, d\r\n"
	}
	
	direction := strings.ToLower(args[0])
	destRoomID := args[1]
	
	// Verify destination exists
	destRoom, err := Manager.GetRoom(destRoomID)
	if err != nil {
		return fmt.Sprintf("Destination room not found: %s\r\n", destRoomID)
	}
	
	// Determine keywords based on direction
	keywords := expandDirection(direction)
	
	// Create exit
	exit := &database.Exit{
		FromRoomID:       player.CurrentRoomID,
		ToRoomID:         destRoomID,
		Keywords:         keywords,
		Description:      fmt.Sprintf("An exit leading %s", direction),
		IsHidden:         false,
		IsObvious:        true,
		AllowLookThrough: true,
		IsOpen:           true,
		IsLocked:         false,
	}
	
	if err := database.CreateExit(exit); err != nil {
		return fmt.Sprintf("Error creating exit: %v\r\n", err)
	}
	
	// Reload room to get new exit
	Manager.ReloadRoom(player.CurrentRoomID)
	
	return fmt.Sprintf("Created exit %s to %s\r\n", direction, destRoom.Title)
}

// CmdExitDelete deletes an exit
func CmdExitDelete(player *Player, args []string) string {
	if len(args) == 0 {
		return "Usage: exit delete <direction>\r\n"
	}
	
	direction := strings.ToLower(args[0])
	
	// Find the exit
	exit, err := Manager.FindExitByKeyword(player.CurrentRoomID, direction)
	if err != nil {
		return fmt.Sprintf("No exit found in direction: %s\r\n", direction)
	}
	
	// Delete it
	if err := database.DeleteExit(exit.ID); err != nil {
		return fmt.Sprintf("Error deleting exit: %v\r\n", err)
	}
	
	// Reload room
	Manager.ReloadRoom(player.CurrentRoomID)
	
	return fmt.Sprintf("Deleted exit %s\r\n", direction)
}

// CmdExitList lists exits from current room
func CmdExitList(player *Player, args []string) string {
	exits, err := Manager.GetAllExits(player.CurrentRoomID)
	if err != nil {
		return fmt.Sprintf("Error loading exits: %v\r\n", err)
	}
	
	if len(exits) == 0 {
		return "No exits from this room.\r\n"
	}
	
	result := "Exits from this room:\r\n"
	for _, exit := range exits {
		keywords := strings.Join(exit.Keywords, ", ")
		destRoom, _ := Manager.GetRoom(exit.ToRoomID)
		destTitle := "Unknown"
		if destRoom != nil {
			destTitle = destRoom.Title
		}
		
		result += fmt.Sprintf("  [%s] -> %s", keywords, destTitle)
		if exit.IsHidden {
			result += " (hidden)"
		}
		if exit.IsLocked {
			result += " (locked)"
		}
		result += "\r\n"
	}
	
	return result
}

// CmdZone handles zone commands
func CmdZone(player *Player, args []string) string {
	if !player.HasKey("keyBuilder") {
		return "You don't have permission to use this command.\r\n"
	}
	
	if len(args) == 0 {
		return "Zone commands:\r\n" +
			"  zone create <name>  - Create a new zone\r\n" +
			"  zone list           - List all zones\r\n"
	}
	
	subCmd := strings.ToLower(args[0])
	subArgs := args[1:]
	
	switch subCmd {
	case "create":
		return CmdZoneCreate(player, subArgs)
	case "list":
		return CmdListZones(player, nil)
	default:
		return fmt.Sprintf("Unknown zone command: %s\r\n", subCmd)
	}
}

// CmdZoneCreate creates a new zone
func CmdZoneCreate(player *Player, args []string) string {
	if len(args) == 0 {
		return "Usage: zone create <name>\r\nExample: zone create The Dark Forest\r\n"
	}
	
	name := strings.Join(args, " ")
	
	zone := &database.Zone{
		Name:        name,
		Description: "A newly created zone.",
		Theme:       "generic",
	}
	
	if err := database.CreateZone(zone); err != nil {
		return fmt.Sprintf("Error creating zone: %v\r\n", err)
	}
	
	return fmt.Sprintf("Created zone: %s\r\nZone ID: %s\r\n", zone.Name, zone.ID)
}

// expandDirection converts a direction shortcut to full keywords
func expandDirection(dir string) []string {
	switch strings.ToLower(dir) {
	case "n", "north":
		return []string{"north", "n"}
	case "s", "south":
		return []string{"south", "s"}
	case "e", "east":
		return []string{"east", "e"}
	case "w", "west":
		return []string{"west", "w"}
	case "ne", "northeast":
		return []string{"northeast", "ne"}
	case "nw", "northwest":
		return []string{"northwest", "nw"}
	case "se", "southeast":
		return []string{"southeast", "se"}
	case "sw", "southwest":
		return []string{"southwest", "sw"}
	case "u", "up":
		return []string{"up", "u"}
	case "d", "down":
		return []string{"down", "d"}
	default:
		return []string{dir}
	}
}