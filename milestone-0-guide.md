# 🏗️ Milestone 0 — Setup Guide (Type It Yourself!)

> **Goal:** Project initialized, repo structure ready, basic server running with `/health` endpoint
> **Time:** ~1-2 hours if you type everything yourself

---

## Step 1: Initialize Go Module

Open your terminal in `B:\Personal-Projects\ByteVault` and run:

```bash
go mod init github.com/adityakkpk/bytevault
```

**What this does:**

- Creates a `go.mod` file (like `package.json` in Node.js)
- `github.com/adityakkpk/bytevault` is your module name — used in all import paths
- Change `adityakkpk` to your actual GitHub username if different

**Check:** You should now see a `go.mod` file. Open it and read it.

---

## Step 2: Create Folder Structure

Create these folders manually (right-click → New Folder, or use terminal):

```
cmd/api/              ← Entry point lives here
internal/config/      ← Config loading
internal/logger/      ← Logging setup
internal/handler/     ← HTTP request handlers
internal/server/      ← Echo server & routes
internal/database/    ← DB connection (used later)
internal/model/       ← Data models (used later)
internal/repository/  ← DB queries (used later)
internal/service/     ← Business logic (used later)
migrations/           ← SQL migrations (used later)
```

Or run this in PowerShell:

```powershell
mkdir cmd/api, internal/config, internal/logger, internal/handler, internal/server, internal/database, internal/model, internal/repository, internal/service, migrations
```

**Why `internal/`?**
Go has a special rule: anything inside `internal/` can ONLY be imported by code within the same module. It's Go's way of making packages private. Outside code can never import your `internal/` packages.

**Why `cmd/api/`?**
Convention. `cmd/` holds your executables. If you had multiple apps (API server, worker, CLI), each gets its own folder under `cmd/`.

---

## Step 3: Your First Go File — `cmd/api/main.go`

Create the file `cmd/api/main.go` and type this:

```go
package main

import "fmt"

func main() {
	fmt.Println("🚀 ByteVault server starting...")
}
```

**Go concepts:**

- `package main` → Every Go executable MUST have a `main` package. This tells Go "this is a runnable program, not a library"
- `import "fmt"` → Importing the `fmt` (format) package from Go's standard library. It has print functions
- `func main()` → THE entry point. Go runs this function when you start the program. Like `if __name__ == "__main__"` in Python
- `fmt.Println()` → Print a line to the terminal. The `P` is uppercase because in Go, **uppercase = exported (public)**

**Run it:**

```bash
go run cmd/api/main.go
```

You should see: `🚀 ByteVault server starting...`

🎉 **Congrats! Your first Go program runs!**

---

## Step 4: Install Dependencies

Run these commands one by one:

```bash
# Echo — HTTP framework (like Express.js)
go get github.com/labstack/echo/v4

# Koanf — Config management (reads .env files & env vars)
go get github.com/knadh/koanf/v2
go get github.com/knadh/koanf/providers/env
go get github.com/knadh/koanf/providers/file
go get github.com/knadh/koanf/parsers/dotenv

# Zerolog — Structured logging (fast, zero allocation)
go get github.com/rs/zerolog
```

**What `go get` does:**

- Downloads the package
- Adds it to `go.mod` (like `npm install` adds to `package.json`)
- Stores downloaded code in `go.sum` (like `package-lock.json`)

**Check:** Open `go.mod` — you'll see all dependencies listed.

---

## Step 5: Create `.env` File

Create `.env` in the project root and type:

```env
SERVER_PORT=8080

DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=bytevault
DB_SSLMODE=disable

APP_ENV=development
```

Also create `.env.example` with the same content (this one gets committed to git as a template).

---

## Step 6: Create `.gitignore`

Create `.gitignore` in the project root:

```gitignore
# Binaries
*.exe
*.exe~
*.dll
*.so
*.dylib
/bin/

# Environment (secrets!)
.env

# IDE
.idea/
.vscode/
*.swp

# OS
.DS_Store
Thumbs.db

# Go
/vendor/
*.test
*.out
coverage.html
coverage.out
```

---

## Step 7: Config Loader — `internal/config/config.go`

Create the file and type this. **Read every comment carefully:**

```go
package config

import (
	"fmt"
	"strings"

	"github.com/knadh/koanf/parsers/dotenv"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// Config is the root config struct.
// A struct groups related data — like a class shape without methods.
// The `koanf:"server"` part is a TAG — it tells Koanf how to map keys.
type Config struct {
	Server   ServerConfig   `koanf:"server"`
	Database DatabaseConfig `koanf:"db"`
	App      AppConfig      `koanf:"app"`
}

type ServerConfig struct {
	Port string `koanf:"port"`
}

type DatabaseConfig struct {
	Host     string `koanf:"host"`
	Port     string `koanf:"port"`
	User     string `koanf:"user"`
	Password string `koanf:"password"`
	Name     string `koanf:"name"`
	SSLMode  string `koanf:"sslmode"`
}

// DSN is a METHOD on DatabaseConfig.
// (d DatabaseConfig) is the RECEIVER — it means "this function belongs to DatabaseConfig"
// It builds a PostgreSQL connection string like:
// postgres://user:pass@host:port/dbname?sslmode=disable
func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.Name, d.SSLMode,
	)
}

type AppConfig struct {
	Env string `koanf:"env"`
}

// Load reads config from .env file + environment variables.
// Returns *Config (pointer) and error.
//
// WHY POINTER (*Config)?
// - Avoids copying the whole struct (performance)
// - Convention: constructors return pointers
// - nil pointer = "no value" (useful for error cases)
func Load() (*Config, error) {
	// koanf.New(".") → "." is the delimiter for nested keys
	// SERVER_PORT → server.port (after our transformation below)
	k := koanf.New(".")

	// Load from .env file first
	if err := k.Load(file.Provider(".env"), dotenv.ParserEnv("", ".", func(s string) string {
		// Transform: SERVER_PORT → server.port
		// 1. strings.ToLower → server_port
		// 2. strings.Replace _ with . → server.port
		return strings.Replace(strings.ToLower(s), "_", ".", -1)
	})); err != nil {
		fmt.Printf("⚠️  No .env file found: %v\n", err)
	}

	// Load from real env vars (overrides .env — important for production!)
	if err := k.Load(env.Provider("", ".", func(s string) string {
		return strings.Replace(strings.ToLower(s), "_", ".", -1)
	}), nil); err != nil {
		return nil, fmt.Errorf("error loading env vars: %w", err)
	}

	// Unmarshal = convert flat key-value map → typed struct
	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, fmt.Errorf("error unmarshalling config: %w", err)
	}

	// &cfg returns a POINTER to cfg
	return &cfg, nil
}
```

**Key Go concepts here:**

- **struct** → Groups related fields (like a TypeScript interface or Python dataclass)
- **struct tags** (`koanf:"port"`) → Metadata that libraries use to map data to fields
- **method receiver** (`func (d DatabaseConfig) DSN()`) → Makes DSN() belong to DatabaseConfig
- **`:=`** → Short variable declaration. Go figures out the type automatically
- **`*Config`** → Pointer to Config. `&cfg` gives you the memory address
- **`if err != nil`** → Go's error handling. No try/catch! Functions return errors explicitly
- **`fmt.Errorf("...: %w", err)`** → Wraps an error with context message

---

## Step 8: Logger — `internal/logger/logger.go`

```go
package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
)

// Log is a package-level variable — any file that imports this package
// can use logger.Log.Info().Msg("hello")
var Log zerolog.Logger

// Init configures the logger based on environment.
// We don't use Go's special init() function because we want
// EXPLICIT control over when this runs (after config is loaded).
func Init(env string) {
	if env == "development" {
		// Pretty colored output for development
		// Instead of JSON: {"level":"info","message":"hello"}
		// You get: 10:30PM INF hello
		output := zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}
		Log = zerolog.New(output).
			With().      // Start adding context fields
			Timestamp(). // Add timestamp to every log
			Caller().    // Add file:line to every log
			Logger()     // Finalize and return the logger
	} else {
		// JSON output for production (for log aggregation tools)
		Log = zerolog.New(os.Stdout).
			With().
			Timestamp().
			Logger()
	}

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
}
```

**Key concepts:**

- **`var Log zerolog.Logger`** → Package-level variable. Accessible as `logger.Log` from other packages
- **Method chaining** → `.With().Timestamp().Caller().Logger()` — each method returns the object so you can chain calls
- **`os.Stdout`** → Standard output (your terminal)

---

## Step 9: Health Handler — `internal/handler/health.go`

```go
package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// HealthHandler holds dependencies for health-related endpoints.
// Empty now, but later we'll add DB, services, etc.
type HealthHandler struct{}

// NewHealthHandler is a CONSTRUCTOR function.
// Go doesn't have constructors, so convention is New<TypeName>()
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// Health handles GET /api/v1/health
// (h *HealthHandler) = this method belongs to HealthHandler
// echo.Context = gives you request data, response helpers
// Returns error (nil = success)
func (h *HealthHandler) Health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]any{
		"status":  "healthy",
		"service": "bytevault",
	})
}
```

**Key concepts:**

- **`echo.Context`** → Interface that gives you request/response access. `c.JSON()` sends JSON
- **`map[string]any{}`** → A map (like JS object). Keys are strings, values can be `any` type
- **`http.StatusOK`** → Constant = 200. From Go's `net/http` standard library
- **Constructor pattern** → `NewHealthHandler()` returns `*HealthHandler` (pointer)

---

## Step 10: Server — `internal/server/server.go`

```go
package server

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/adityakkpk/bytevault/internal/config"
	"github.com/adityakkpk/bytevault/internal/logger"
)

// Server holds the Echo instance and all dependencies.
type Server struct {
	echo   *echo.Echo
	config *config.Config
}

// New creates a configured server. This is dependency injection:
// we PASS the config in, rather than reading it from a global.
func New(cfg *config.Config) *Server {
	e := echo.New()
	e.HideBanner = true

	// Middleware = functions that run BEFORE your handler
	// Request → Recover → CORS → RequestID → Your Handler → Response

	e.Use(middleware.Recover())  // Catches panics, returns 500 instead of crash
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
	}))
	e.Use(middleware.RequestID()) // Adds unique ID to each request

	s := &Server{
		echo:   e,
		config: cfg,
	}

	s.registerRoutes()

	return s
}

// Start begins listening. This BLOCKS until server shuts down.
func (s *Server) Start() error {
	port := s.config.Server.Port
	if port == "" {
		port = "8080"
	}

	logger.Log.Info().
		Str("port", port).
		Str("env", s.config.App.Env).
		Msg("🚀 ByteVault server starting")

	return s.echo.Start(":" + port)
}
```

**Key concepts:**

- **Struct fields** → `echo *echo.Echo` means the Server struct HAS an Echo instance
- **Lowercase field names** (`echo`, `config`) → Private/unexported. Only this package can access them
- **Dependency injection** → Config is passed in via `New(cfg)`, not read from global state
- **Middleware chain** → Each request passes through all middleware before reaching your handler

---

## Step 11: Routes — `internal/server/routes.go`

```go
package server

import (
	"github.com/adityakkpk/bytevault/internal/handler"
)

// registerRoutes sets up all API endpoints.
func (s *Server) registerRoutes() {
	healthHandler := handler.NewHealthHandler()

	// Group = route prefix. All routes below start with /api/v1
	v1 := s.echo.Group("/api/v1")

	v1.GET("/health", healthHandler.Health)
}
```

**Key concepts:**

- **Same package** → This file is also `package server`, so it can access `Server` struct
- **Route groups** → `s.echo.Group("/api/v1")` prefixes all routes with `/api/v1`
- **`v1.GET("/health", handler)`** → Maps GET /api/v1/health to the Health function

---

## Step 12: Update `cmd/api/main.go`

Replace your simple main.go with the full version:

```go
package main

import (
	"os"

	"github.com/adityakkpk/bytevault/internal/config"
	"github.com/adityakkpk/bytevault/internal/logger"
	"github.com/adityakkpk/bytevault/internal/server"
)

func main() {
	// Step 1: Load config from .env + environment variables
	cfg, err := config.Load()
	if err != nil {
		os.Stderr.WriteString("❌ Failed to load config: " + err.Error() + "\n")
		os.Exit(1)
	}

	// Step 2: Initialize logger
	logger.Init(cfg.App.Env)
	logger.Log.Info().Msg("✅ Configuration loaded")

	// Step 3: Create and start server
	srv := server.New(cfg)
	if err := srv.Start(); err != nil {
		logger.Log.Fatal().Err(err).Msg("❌ Server failed")
	}
}
```

---

## Step 13: Tidy & Run

```bash
# Clean up unused deps, add missing ones
go mod tidy

# Run the server
go run cmd/api/main.go
```

You should see logs showing the server started on port 8080.

**Test the health endpoint** (open another terminal):

```powershell
Invoke-WebRequest -Uri "http://localhost:8080/api/v1/health" -UseBasicParsing | Select-Object -ExpandProperty Content
```

Expected output: `{"service":"bytevault","status":"healthy"}`

---

## Step 14: Create Makefile (Optional but Handy)

Create `Makefile` in the root:

```makefile
.PHONY: run build test clean tidy

run:
	go run cmd/api/main.go

build:
	go build -o bin/bytevault.exe cmd/api/main.go

test:
	go test ./... -v

clean:
	del /Q bin\* 2>nul || true

tidy:
	go mod tidy
```

Now you can use `make run` instead of typing the full command.

---

## ✅ Milestone 0 Complete When:

- [ ] `go run cmd/api/main.go` starts without errors
- [ ] `GET /api/v1/health` returns `{"status":"healthy","service":"bytevault"}`
- [ ] You understand what every line does

---

## 📝 Go Concepts You Learned in Milestone 0

| Concept              | What It Is                                                |
| -------------------- | --------------------------------------------------------- |
| `package main`       | Entry point package for executables                       |
| `func main()`        | The function Go runs first                                |
| `struct`             | Groups related data (like a class shape)                  |
| `method receiver`    | `func (s *Server) Start()` — function belongs to a struct |
| `pointer *`          | References memory address. `&x` = address of x            |
| `:=`                 | Short declaration — Go infers the type                    |
| `if err != nil`      | Go's error handling pattern (no try/catch)                |
| `import`             | Grouped: stdlib → external → internal                     |
| `internal/`          | Private packages — can't be imported externally           |
| `Uppercase = Public` | `Health()` is exported, `echo` field is not               |

---

## What's Next: Milestone 0.5 — Database & Auth

Tell me when you're done with Milestone 0 and we'll start:

1. PostgreSQL connection with pgx
2. Tern migrations (users table)
3. User registration & JWT tokens
