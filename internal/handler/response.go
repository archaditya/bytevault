package handler

import (
	"github.com/labstack/echo/v4"
)

// APIResponse represents the standardized JSON envelope for all API responses.
type APIResponse struct {
	Status     string `json:"status"`
	Detail     string `json:"detail,omitempty"`
	Data       any    `json:"data,omitempty"`
	Pagination any    `json:"pagination,omitempty"`
}

// PaginationMetadata represents standard cursor or offset pagination metadata.
type PaginationMetadata struct {
	Total      int `json:"total,omitempty"`
	Limit      int `json:"limit,omitempty"`
	Page       int `json:"page,omitempty"`
	NextCursor string `json:"next_cursor,omitempty"`
}

// SendSuccess sends a standardized 2xx JSON response.
func SendSuccess(c echo.Context, statusCode int, data any, pagination any) error {
	return c.JSON(statusCode, APIResponse{
		Status:     "success",
		Data:       data,
		Pagination: pagination,
	})
}

// SendError sends a standardized error JSON response.
func SendError(c echo.Context, statusCode int, errorMessage string) error {
	return c.JSON(statusCode, APIResponse{
		Status: "error",
		Detail: errorMessage,
	})
}
