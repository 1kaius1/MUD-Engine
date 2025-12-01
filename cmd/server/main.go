package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"mudengine/internal/config"
)

// AuthState represents the current authentication state of a connection
type AuthState int

const (
	StateConnected AuthState = iota
	StateAwaitingLogin
	StateAwaitingPassword
	StateAwaitingMFA
	StateAuthenticated
)

// Client represents a connected player
type Client struct {
	conn          *websocket.Conn
	send          chan []byte
	authState     AuthState
	username      string
	failedAttempts int
	mu            sync.Mutex
}

// Server manages all connected clients
type Server struct {
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	shutdown   chan struct{}
	mu         sync.RWMutex
}

// WebSocket upgrader configuration
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// TODO: In production, validate origin properly
		return true
	},
}

// NewServer creates a new server instance
func NewServer() *Server {
	return &Server{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		shutdown:   make(chan struct{}),
	}
}

// Run starts the server's main event loop
func (s *Server) Run() {
	for {
		select {
		case client := <-s.register:
			s.mu.Lock()
			s.clients[client] = true
			s.mu.Unlock()
			log.Printf("Client connected. Total clients: %d", len(s.clients))

		case client := <-s.unregister:
			s.mu.Lock()
			if _, ok := s.clients[client]; ok {
				delete(s.clients, client)
				close(client.send)
				log.Printf("Client disconnected. Total clients: %d", len(s.clients))
			}
			s.mu.Unlock()
			
		case <-s.shutdown:
			log.Println("Server shutting down, closing all client connections...")
			s.mu.Lock()
			for client := range s.clients {
				client.sendMessage("\r\n\r\nServer is shutting down. Goodbye!\r\n")
				client.conn.Close()
				close(client.send)
			}
			s.clients = make(map[*Client]bool)
			s.mu.Unlock()
			log.Println("All clients disconnected.")
			return
		}
	}
}

// handleWebSocket handles incoming WebSocket connections
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	client := &Client{
		conn:      conn,
		send:      make(chan []byte, 256),
		authState: StateConnected,
	}

	s.register <- client

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump(s)
}

// readPump reads messages from the WebSocket connection
func (c *Client) readPump(s *Server) {
	defer func() {
		s.unregister <- c
		c.conn.Close()
	}()

	// Set read deadline and pong handler
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Send welcome banner
	c.sendWelcomeBanner()

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Process the message based on authentication state
		c.processMessage(string(message))
	}
}

// writePump writes messages to the WebSocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current write
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// sendWelcomeBanner sends the initial banner and login prompt
func (c *Client) sendWelcomeBanner() {
	banner := `
╔════════════════════════════════════════╗
║     Welcome to the MUD Server v0.1     ║
║                                        ║
║     A Text-Based Adventure Awaits      ║
╔════════════════════════════════════════╝

`
	c.sendMessage(banner)
	c.mu.Lock()
	c.authState = StateAwaitingLogin
	c.mu.Unlock()
	c.sendMessage("Login: ")
}

// processMessage handles incoming messages based on authentication state
func (c *Client) processMessage(message string) {
	c.mu.Lock()
	state := c.authState
	c.mu.Unlock()

	switch state {
	case StateAwaitingLogin:
		c.handleLogin(message)
	case StateAwaitingPassword:
		c.handlePassword(message)
	case StateAwaitingMFA:
		c.handleMFA(message)
	case StateAuthenticated:
		c.handleGameCommand(message)
	default:
		c.sendMessage("Error: Invalid state\r\n")
	}
}

// handleLogin processes the login username
func (c *Client) handleLogin(username string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if username == "" {
		c.sendMessage("Login cannot be empty.\r\nLogin: ")
		return
	}

	// TODO: Validate username format
	c.username = username
	c.authState = StateAwaitingPassword
	c.sendMessage("Password: \x1b[8m") // ANSI code to hide input
}

// handlePassword processes the password
func (c *Client) handlePassword(password string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.sendMessage("\x1b[28m") // ANSI code to show input again

	if password == "" {
		c.sendMessage("Password cannot be empty.\r\nPassword: \x1b[8m")
		return
	}

	// TODO: Validate password against database
	// For now, accept any non-empty password
	isValid := c.validatePassword(password)

	if !isValid {
		c.failedAttempts++
		if c.failedAttempts >= 3 {
			c.sendMessage("Too many failed attempts. Disconnecting.\r\n")
			c.conn.Close()
			return
		}
		c.sendMessage(fmt.Sprintf("Invalid credentials. Attempts remaining: %d\r\nLogin: ", 3-c.failedAttempts))
		c.authState = StateAwaitingLogin
		c.username = ""
		return
	}

	c.authState = StateAwaitingMFA
	c.sendMessage("MFA Code: ")
}

// handleMFA processes the MFA code
func (c *Client) handleMFA(code string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if code == "" {
		c.sendMessage("MFA code cannot be empty.\r\nMFA Code: ")
		return
	}

	// TODO: Validate TOTP code
	// For now, accept "123456" as valid
	isValid := c.validateMFA(code)

	if !isValid {
		c.failedAttempts++
		if c.failedAttempts >= 3 {
			c.sendMessage("Too many failed attempts. Disconnecting.\r\n")
			c.conn.Close()
			return
		}
		c.sendMessage(fmt.Sprintf("Invalid MFA code. Attempts remaining: %d\r\nMFA Code: ", 3-c.failedAttempts))
		return
	}

	c.authState = StateAuthenticated
	c.sendMessage(fmt.Sprintf("\r\nWelcome back, %s!\r\n\r\n", c.username))
	
	// TODO: Load player's current room from database
	// For now, show a default room description
	c.sendInitialLook()
	
	c.sendMessage("> ")
}

// sendInitialLook sends the room description when player first logs in
func (c *Client) sendInitialLook() {
	// TODO: Replace with actual room data from database
	// This is placeholder content until we implement the room system
	c.sendMessage("The Town Square\r\n")
	c.sendMessage("You stand in the bustling town square. A large fountain dominates\r\n")
	c.sendMessage("the center, with merchants hawking their wares around its edge.\r\n")
	c.sendMessage("A weathered wooden sign stands near the fountain.\r\n\r\n")
	c.sendMessage("Obvious exits: north, south, east\r\n")
	c.sendMessage("You see: a weathered wooden sign\r\n\r\n")
}

// handleGameCommand processes authenticated game commands
func (c *Client) handleGameCommand(command string) {
	switch command {
	case "look":
		c.sendMessage("You are in a dimly lit room. There is a door to the north.\r\n> ")
	case "quit":
		c.sendMessage("Goodbye!\r\n")
		c.conn.Close()
	default:
		c.sendMessage(fmt.Sprintf("Unknown command: %s\r\n> ", command))
	}
}

// validatePassword validates the password (placeholder)
func (c *Client) validatePassword(password string) bool {
	// TODO: Implement actual password validation with bcrypt
	// For now, accept any password for user "admin"
	return c.username == "admin" && password == "password"
}

// validateMFA validates the MFA code (placeholder)
func (c *Client) validateMFA(code string) bool {
	// TODO: Implement actual TOTP validation
	// For now, accept "123456"
	return code == "123456"
}

// Shutdown initiates graceful shutdown
func (s *Server) Shutdown() {
	close(s.shutdown)
}

// sendMessage sends a message to the client
func (c *Client) sendMessage(message string) {
	select {
	case c.send <- []byte(message):
	default:
		// Channel full, client too slow
		log.Printf("Client send buffer full for %s", c.username)
	}
}

const (
	ServerVersion = "0.1.0"
	ServerName    = "MUD Engine"
)

func main() {
	// Load configuration from .env file
	// Use -env flag to specify custom file: go run main.go -env custom.env
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	
	// Log configuration
	cfg.LogConfig()
	
	log.Printf("%s v%s starting up...", cfg.ServerName, cfg.ServerVersion)
	
	server := NewServer()
	go server.Run()

	// HTTP handlers
	http.HandleFunc("/ws", server.handleWebSocket)
	
	// Serve static files for web client
	// This serves all files from web/static directory
	// index.html will be served by default for "/"
	fs := http.FileServer(http.Dir("web/static"))
	http.Handle("/", fs)

	// Create HTTP server with timeouts
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.ServerPort),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Set up graceful shutdown on SIGINT (Ctrl+C) or SIGTERM
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start HTTP server in a goroutine
	go func() {
		log.Printf("%s v%s ready", cfg.ServerName, cfg.ServerVersion)
		log.Printf("WebSocket endpoint: ws://localhost:%d/ws", cfg.ServerPort)
		log.Printf("Web client: http://localhost:%d/", cfg.ServerPort)
		log.Println("Press Ctrl+C to shutdown")
		
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	sig := <-sigChan
	log.Printf("\nReceived signal: %v", sig)
	performGracefulShutdown(server, httpServer, cfg)
}

// performGracefulShutdown handles the shutdown sequence
func performGracefulShutdown(server *Server, httpServer *http.Server, cfg *config.Config) {
	log.Printf("%s v%s shutting down...", cfg.ServerName, cfg.ServerVersion)
	
	// Step 1: Stop accepting new connections
	log.Println("[1/5] Stopping new connections...")
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.ShutdownTimeoutSecs)*time.Second)
	defer cancel()
	
	// Step 2: Notify all connected players
	log.Println("[2/5] Notifying connected players...")
	server.Shutdown() // This sends messages to clients and closes connections
	
	// Step 3: Save all player data
	log.Println("[3/5] Saving player data...")
	// TODO: Save all authenticated players' locations and status to database
	saveAllPlayerData(server)
	time.Sleep(500 * time.Millisecond) // Simulate database writes
	
	// Step 4: Flush pending database writes
	log.Println("[4/5] Flushing database writes...")
	// TODO: Ensure all database transactions are committed
	flushDatabaseWrites()
	time.Sleep(500 * time.Millisecond) // Simulate flush
	
	// Step 5: Shutdown HTTP server
	log.Println("[5/5] Shutting down HTTP server...")
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}
	
	log.Printf("%s v%s offline.", cfg.ServerName, cfg.ServerVersion)
}

// saveAllPlayerData saves all connected players' current state
func saveAllPlayerData(server *Server) {
	// TODO: Implement when we have database layer
	server.mu.RLock()
	defer server.mu.RUnlock()
	
	playerCount := 0
	for client := range server.clients {
		if client.authState == StateAuthenticated {
			// TODO: Save player location, health, inventory, etc.
			// For now, just log
			log.Printf("  - Saving player: %s", client.username)
			playerCount++
		}
	}
	
	if playerCount > 0 {
		log.Printf("  Saved %d player(s)", playerCount)
	} else {
		log.Println("  No authenticated players to save")
	}
}

// flushDatabaseWrites ensures all pending database writes complete
func flushDatabaseWrites() {
	// TODO: Implement when we have database layer
	// This should:
	// - Commit any pending transactions
	// - Sync to disk
	// - Close database connections cleanly
	log.Println("  Database flush complete")
}

/*
================================================================================
PROJECT ROADMAP - TODO LIST
================================================================================

PHASE 1 - CORE AUTHENTICATION & SECURITY
[ ] Implement TLS/SSL certificates (Let's Encrypt or self-signed for dev)
[ ] Migrate from HTTP to HTTPS (WebSocket over TLS - wss://)
[ ] Implement bcrypt password hashing and validation
[ ] Implement TOTP/MFA validation using github.com/pquerna/otp
[ ] Add proper WebSocket origin checking for security
[ ] Implement rate limiting for authentication attempts
[ ] Add IP-based temporary bans after repeated failures
[ ] Implement account lockout after X failed attempts
[ ] Add authentication attempt logging (timestamp, IP, success/failure)

PHASE 2 - DATABASE LAYER (SQLite)
[ ] Design database schema (users, characters, rooms, items, etc.)
[ ] Create database initialization and migration system
[ ] Implement user registration system
[ ] Implement password storage with bcrypt
[ ] Implement MFA secret storage and setup flow
[ ] Add QR code generation for MFA enrollment
[ ] Implement session management
[ ] Add character creation and storage
[ ] Implement player inventory system
[ ] Add room/world data storage

PHASE 3 - GAME ENGINE CORE
[ ] Design and implement command parser with abbreviations
[ ] Implement room system with exits and descriptions
[ ] Add zone/area management
[ ] Implement player movement between rooms
[ ] Add "look" command with detailed room descriptions
[ ] Implement "say" and "emote" commands
[ ] Add room event broadcasting to all players in room
[ ] Implement player-to-player communication (tell/whisper)
[ ] Add global chat channels
[ ] Implement game loop/ticker for periodic updates

PHASE 4 - WEB CLIENT
[ ] Create HTML/CSS/JS web client
[ ] Implement WebSocket connection from browser
[ ] Add ANSI color code rendering (ansi-to-html or similar)
[ ] Create scrolling terminal display
[ ] Implement password input masking
[ ] Add command history (up/down arrow navigation)
[ ] Make mobile-friendly with touch controls
[ ] Add common command buttons for mobile
[ ] Implement graceful reconnection handling
[ ] Add connection status indicators
[ ] Implement client capability negotiation protocol
[ ] Add status bar for HP/MP/conditions (Phase 1: text display)
[ ] Implement JSON protocol for status updates
[ ] Add /statusbar command to toggle display
[ ] Create visual progress bars for HP/MP (Phase 2)
[ ] Add color coding for status (red=low HP, etc.)
[ ] Implement status condition display (poisoned, haste, etc.)
[ ] Add auto-hide status bars on mobile/small screens
[ ] Implement GMCP (Generic Mud Communication Protocol) support (Phase 3)
[ ] Add configurable status bar layouts
[ ] Implement server capability detection and fallback prompts

PHASE 5 - GAME FEATURES
[ ] Implement item system (objects in world and inventory)
[ ] Add "get", "drop", "give" commands
[ ] Implement "examine" for detailed item descriptions
[ ] Add container objects (bags, chests, etc.)
[ ] Implement NPC (Non-Player Character) system
[ ] Add basic NPC dialogue system
[ ] Implement combat system (turns, damage, health)
[ ] Add experience points and leveling
[ ] Implement character stats (strength, dexterity, etc.)
[ ] Add equipment system (wear/wield items)

PHASE 6 - ADVANCED FEATURES
[ ] Implement quest system
[ ] Add crafting/profession system
[ ] Implement merchant/shop system
[ ] Add player housing/personal spaces
[ ] Implement guild/clan system
[ ] Add mail system for offline communication
[ ] Implement bulletin board system
[ ] Add save/backup system for world state

PHASE 7 - REDIS INTEGRATION
[ ] Add Redis connection and configuration
[ ] Implement session caching in Redis
[ ] Cache frequently accessed room data
[ ] Add player online status tracking
[ ] Implement real-time metrics and statistics
[ ] Add pub/sub for cross-server communication (if multi-server)
[ ] Cache leaderboards and rankings

PHASE 8 - POSTGRESQL MIGRATION
[ ] Design PostgreSQL schema migration from SQLite
[ ] Implement database abstraction layer
[ ] Create migration scripts for existing data
[ ] Update all database queries for PostgreSQL
[ ] Implement connection pooling
[ ] Add prepared statements for performance
[ ] Implement database backup strategies

PHASE 9 - OPERATIONS & MONITORING
[ ] Add structured logging (logrus or zap)
[ ] Implement metrics collection (Prometheus)
[ ] Add health check endpoints
[ ] Implement graceful shutdown
[ ] Add configuration management (environment variables, config files)
[ ] Implement hot-reload for configuration changes
[ ] Add admin commands and web admin panel
[ ] Implement player reporting and moderation tools

PHASE 10 - SCALING & OPTIMIZATION
[ ] Profile and optimize hot paths
[ ] Implement object pooling for common allocations
[ ] Add caching strategies for expensive operations
[ ] Optimize database queries with indexes
[ ] Implement horizontal scaling architecture
[ ] Add load balancing support
[ ] Implement distributed game state (if multi-server)
[ ] Add CDN support for static assets

PHASE 11 - CONTENT & POLISH
[ ] Create comprehensive help system
[ ] Write tutorial/newbie area
[ ] Add ANSI color themes
[ ] Implement sound/music triggers (for supporting clients)
[ ] Add achievements system
[ ] Implement world events and dynamic content
[ ] Add seasonal events
[ ] Create admin content creation tools

PHASE 12 - TESTING & DOCUMENTATION
[ ] Write unit tests for core systems
[ ] Add integration tests
[ ] Implement load testing
[ ] Write API documentation
[ ] Create player documentation/wiki
[ ] Add builder/admin documentation
[ ] Create deployment documentation

PHASE 13 - LEGACY COMPATIBILITY (LOWEST PRIORITY)
[ ] Implement raw TCP/Telnet server on separate port (e.g., 4000)
[ ] Add Telnet protocol negotiation (IAC, WILL, WONT, DO, DONT)
[ ] Implement Telnet option: ECHO (for password masking)
[ ] Implement Telnet option: NAWS (Negotiate About Window Size)
[ ] Share authentication and game logic with WebSocket server
[ ] Add configuration option to enable/disable Telnet server
[ ] Document Telnet client compatibility (PuTTY, MUSHclient, etc.)
[ ] Test with popular MUD clients (Mudlet, TinTin++, MUSHclient)

================================================================================
CURRENT PHASE: Phase 1 - Core Authentication & Security
NEXT MILESTONE: Complete web client and basic room system
================================================================================
*/