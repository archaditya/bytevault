// Package handler contains HTTP request handlers.
//
// WHAT IS A HANDLER?
// A handler is a function that processes an HTTP request and returns a response.
// In Echo, every handler has the signature: func(c echo.Context) error
//
// echo.Context gives you access to:
//   - c.Request()  → the raw HTTP request
//   - c.Response() → the response writer
//   - c.JSON()     → send JSON response
//   - c.Param()    → URL path parameters
//   - c.Bind()     → parse request body into a struct
//
// GO CONCEPTS:
// - map[string]interface{} — A map where keys are strings and values can be ANY type.
//   "interface{}" in Go means "any type" (similar to TypeScript's `any`).
//   In newer Go (1.18+), you can also write `map[string]any`.
package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// HealthHandler handles health check requests.
// This is a struct that will hold dependencies (like DB, services) later.
// For now it's empty, but using a struct lets us add dependencies without
// changing the handler function signatures.
type HealthHandler struct{}

// NewHealthHandler creates a new HealthHandler.
// This is a CONSTRUCTOR FUNCTION — Go doesn't have constructors,
// so by convention we create New<Type>() functions.
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// Health returns the server health status.
// The (h *HealthHandler) part is called a METHOD RECEIVER.
// It means this function "belongs to" HealthHandler.
// You call it like: handler.Health(c)
func (h *HealthHandler) Health(c echo.Context) error {
	// c.JSON takes:
	// 1. HTTP status code (200 = OK)
	// 2. Any value to serialize as JSON
	return c.JSON(http.StatusOK, map[string]any{
		"status":  "healthy",
		"service": "bytevault",
	})
}
