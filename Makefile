# ============================================
# ByteVault Makefile
# ============================================
# A Makefile gives you shortcut commands.
# Instead of typing "go run cmd/api/main.go",
# you just type "make run".
#
# USAGE:
#   make run      → Start the dev server
#   make build    → Compile to binary
#   make test     → Run all tests
#   make clean    → Remove build artifacts
# ============================================

.PHONY: run build test clean

# Run the server in development mode
run:
	go run cmd/api/main.go

# Build a production binary
build:
	go build -o bin/bytevault.exe cmd/api/main.go

# Run all tests
test:
	go test ./... -v

# Clean build artifacts
clean:
	del /Q bin\* 2>nul || true

# Tidy dependencies (remove unused, add missing)
tidy:
	go mod tidy
