package handler

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/archaditya/bytevault/internal/model"
	"github.com/archaditya/bytevault/internal/service"
)

type ContactHandler struct {
	contactService *service.ContactService
}

func NewContactHandler(contactService *service.ContactService) *ContactHandler {
	return &ContactHandler{contactService: contactService}
}

func (h *ContactHandler) Submit(c echo.Context) error {
	var req struct {
		Name    string `json:"name"`
		Email   string `json:"email"`
		Subject string `json:"subject"`
		Message string `json:"message"`
	}

	if err := c.Bind(&req); err != nil {
		return SendError(c, http.StatusBadRequest, "Invalid request body")
	}

	q := &model.ContactQuery{
		Name:    req.Name,
		Email:   req.Email,
		Subject: req.Subject,
		Message: req.Message,
	}

	if err := h.contactService.SubmitQuery(c.Request().Context(), q); err != nil {
		return SendError(c, http.StatusBadRequest, err.Error())
	}

	return SendSuccess(c, http.StatusCreated, q, nil)
}

func (h *ContactHandler) List(c echo.Context) error {
	page, _ := strconv.Atoi(c.QueryParam("page"))
	limit, _ := strconv.Atoi(c.QueryParam("limit"))

	if page < 1 { page = 1 }
	if limit < 1 { limit = 20 }
	offset := (page - 1) * limit

	queries, total, err := h.contactService.ListQueries(c.Request().Context(), limit, offset)
	if err != nil {
		return SendError(c, http.StatusInternalServerError, "Failed to retrieve queries")
	}

	pagination := PaginationMetadata{
		Total: total,
		Limit: limit,
		Page:  page,
	}

	return SendSuccess(c, http.StatusOK, map[string]any{"queries": queries}, pagination)
}

func (h *ContactHandler) Reply(c echo.Context) error {
	id := c.Param("id")
	var req struct {
		Reply string `json:"reply"`
	}

	if err := c.Bind(&req); err != nil {
		return SendError(c, http.StatusBadRequest, "Invalid request body")
	}

	repliedBy := c.Get("user_id").(string)

	if err := h.contactService.ReplyToQuery(c.Request().Context(), id, req.Reply, repliedBy); err != nil {
		return SendError(c, http.StatusInternalServerError, err.Error())
	}

	return SendSuccess(c, http.StatusOK, map[string]string{"message": "Reply saved successfully"}, nil)
}
