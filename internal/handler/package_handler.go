package handler

import (
	"net/http"
	"strconv"

	"github.com/archaditya/bytevault/internal/model"
	"github.com/archaditya/bytevault/internal/repository"
	"github.com/archaditya/bytevault/internal/service"
	"github.com/labstack/echo/v4"
)

type PackageHandler struct {
	pkgService    *service.PackageService
	subService    *service.SubscriptionService
	txnService    *service.TransactionService
	subRepo       *repository.SubscriptionRepository
	auditRepo     *repository.SubscriptionAuditRepository
	systemLogRepo *repository.SystemLogRepository
}

func NewPackageHandler(
	pkgService *service.PackageService,
	subService *service.SubscriptionService,
	txnService *service.TransactionService,
	subRepo *repository.SubscriptionRepository,
	auditRepo *repository.SubscriptionAuditRepository,
) *PackageHandler {
	return &PackageHandler{
		pkgService: pkgService,
		subService: subService,
		txnService: txnService,
		subRepo:    subRepo,
		auditRepo:  auditRepo,
	}
}

func (h *PackageHandler) SetSystemLogRepo(r *repository.SystemLogRepository) {
	h.systemLogRepo = r
}

// ListPackages returns all active packages for public display (GET /api/v1/packages).
func (h *PackageHandler) ListPackages(c echo.Context) error {
	packages, err := h.pkgService.ListPackages(c.Request().Context(), true)
	if err != nil {
		return SendError(c, http.StatusInternalServerError, err.Error())
	}
	return SendSuccess(c, http.StatusOK, packages, nil)
}

// GetPackage returns details for a single package (GET /api/v1/packages/:id).
func (h *PackageHandler) GetPackage(c echo.Context) error {
	id := c.Param("id")
	pkg, err := h.pkgService.GetPackageByID(c.Request().Context(), id)
	if err != nil {
		return SendError(c, http.StatusNotFound, "package not found")
	}
	return SendSuccess(c, http.StatusOK, pkg, nil)
}

// AdminCreatePackage handles package creation and Razorpay plan sync (POST /api/v1/admin/packages).
func (h *PackageHandler) AdminCreatePackage(c echo.Context) error {
	var pkg model.Package
	if err := c.Bind(&pkg); err != nil || pkg.Name == "" || pkg.DisplayName == "" {
		return SendError(c, http.StatusBadRequest, "invalid package data: name and display_name are required")
	}

	if pkg.Currency == "" {
		pkg.Currency = "INR"
	}
	if pkg.BillingPeriod == "" {
		pkg.BillingPeriod = "monthly"
	}

	created, err := h.pkgService.CreatePackage(c.Request().Context(), &pkg)
	if err != nil {
		return SendError(c, http.StatusBadRequest, err.Error())
	}

	return SendSuccess(c, http.StatusCreated, created, nil)
}

// AdminUpdatePackage handles package updates (PUT /api/v1/admin/packages/:id).
func (h *PackageHandler) AdminUpdatePackage(c echo.Context) error {
	id := c.Param("id")
	var pkg model.Package
	if err := c.Bind(&pkg); err != nil {
		return SendError(c, http.StatusBadRequest, "invalid package data")
	}
	pkg.ID = id

	if err := h.pkgService.UpdatePackage(c.Request().Context(), &pkg); err != nil {
		return SendError(c, http.StatusBadRequest, err.Error())
	}

	return SendSuccess(c, http.StatusOK, pkg, nil)
}

// AdminDeletePackage deactivates or removes a package (DELETE /api/v1/admin/packages/:id).
func (h *PackageHandler) AdminDeletePackage(c echo.Context) error {
	id := c.Param("id")
	if err := h.pkgService.DeletePackage(c.Request().Context(), id); err != nil {
		return SendError(c, http.StatusBadRequest, err.Error())
	}
	return SendSuccess(c, http.StatusOK, map[string]string{"message": "package deleted"}, nil)
}

// AdminListSubscriptions lists all subscriptions with optional status filter (GET /api/v1/admin/subscriptions).
func (h *PackageHandler) AdminListSubscriptions(c echo.Context) error {
	status := c.QueryParam("status")
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset, _ := strconv.Atoi(c.QueryParam("offset"))
	if offset < 0 {
		offset = 0
	}

	subs, total, err := h.subRepo.ListAll(c.Request().Context(), status, limit, offset)
	if err != nil {
		return SendError(c, http.StatusInternalServerError, err.Error())
	}

	return SendSuccess(c, http.StatusOK, subs, PaginationMetadata{
		Total: total,
		Limit: limit,
	})
}

// AdminCancelSubscription cancels a subscription immediately (POST /api/v1/admin/subscriptions/:id/cancel).
func (h *PackageHandler) AdminCancelSubscription(c echo.Context) error {
	id := c.Param("id")
	if err := h.subService.AdminCancel(c.Request().Context(), id); err != nil {
		return SendError(c, http.StatusBadRequest, err.Error())
	}
	return SendSuccess(c, http.StatusOK, map[string]string{"message": "subscription cancelled"}, nil)
}

// AdminAssignSubscription directly assigns a package to a user (POST /api/v1/admin/users/:id/assign-subscription).
func (h *PackageHandler) AdminAssignSubscription(c echo.Context) error {
	userID := c.Param("id")

	var req struct {
		PackageID    string `json:"package_id"`
		DurationDays int    `json:"duration_days"`
	}
	if err := c.Bind(&req); err != nil || req.PackageID == "" {
		return SendError(c, http.StatusBadRequest, "package_id is required")
	}
	if req.DurationDays <= 0 {
		req.DurationDays = 30
	}

	sub, err := h.subService.AdminAssign(c.Request().Context(), userID, req.PackageID, req.DurationDays)
	if err != nil {
		return SendError(c, http.StatusBadRequest, err.Error())
	}

	return SendSuccess(c, http.StatusOK, sub, nil)
}

// AdminListTransactions lists all transactions across users (GET /api/v1/admin/transactions).
func (h *PackageHandler) AdminListTransactions(c echo.Context) error {
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset, _ := strconv.Atoi(c.QueryParam("offset"))
	if offset < 0 {
		offset = 0
	}

	txns, total, err := h.txnService.ListAllTransactions(c.Request().Context(), limit, offset)
	if err != nil {
		return SendError(c, http.StatusInternalServerError, err.Error())
	}

	return SendSuccess(c, http.StatusOK, txns, PaginationMetadata{
		Total: total,
		Limit: limit,
	})
}

// AdminListAuditLogs lists subscription audit log events (GET /api/v1/admin/subscription-logs).
func (h *PackageHandler) AdminListAuditLogs(c echo.Context) error {
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	offset, _ := strconv.Atoi(c.QueryParam("offset"))
	if offset < 0 {
		offset = 0
	}

	logs, total, err := h.auditRepo.ListAll(c.Request().Context(), limit, offset)
	if err != nil {
		return SendError(c, http.StatusInternalServerError, err.Error())
	}

	return SendSuccess(c, http.StatusOK, logs, PaginationMetadata{
		Total: total,
		Limit: limit,
	})
}

// AdminListSystemLogs returns tracked daily system log archives across local server and R2 (GET /api/v1/admin/system-logs).
func (h *PackageHandler) AdminListSystemLogs(c echo.Context) error {
	if h.systemLogRepo == nil {
		return SendSuccess(c, http.StatusOK, []*model.SystemLogArchive{}, PaginationMetadata{})
	}

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	offset, _ := strconv.Atoi(c.QueryParam("offset"))
	if offset < 0 {
		offset = 0
	}

	logs, total, err := h.systemLogRepo.ListAll(c.Request().Context(), limit, offset)
	if err != nil {
		return SendError(c, http.StatusInternalServerError, err.Error())
	}

	return SendSuccess(c, http.StatusOK, logs, PaginationMetadata{
		Total: total,
		Limit: limit,
	})
}
