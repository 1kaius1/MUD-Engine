# MUD Engine
## Directory Structure Explanation
```
mud-project/
├── cmd/
│   └── server/          # Main application entry point
│       └── main.go
├── internal/            # Private application code
│   ├── auth/            # Authentication logic
│   ├── config/          # Configuration module
│       └── config.go
│   ├── database/        # Database layer
│   ├── game/            # Game logic, rooms, commands
│   ├── player/          # Player management
│   └── websocket/       # WebSocket handling
├── web/
│   └── static/          # HTML/CSS/JS for web client
├── data/                # SQLite database files, configs
├── go.mod               # Go module dependencies
└── go.sum               # Dependency checksums
```

## Essential Go Packages
```
# WebSocket support
go get github.com/gorilla/websocket

# SQLite driver
go get github.com/mattn/go-sqlite3

# PostgreSQL driver (for later)
go get github.com/lib/pq

# Password hashing
go get golang.org/x/crypto/bcrypt

# TOTP/MFA support
go get github.com/pquerna/otp

# Environment variable management
go get github.com/joho/godotenv

# Redis client (for later)
go get github.com/redis/go-redis/v9
```

## Helpful development tools
```
# Install gopls (Go language server for IDE support)
go install golang.org/x/tools/gopls@latest

# Install staticcheck (linter)
go install honnef.co/go/tools/cmd/staticcheck@latest

# Install gofumpt (formatter)
go install mvdan.cc/gofumpt@latest
```

## Test the Engine
In the top project directory the following line will start the game engine for local testing:
```
go run cmd/server/main.go
```

