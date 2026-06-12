package handler

import (
	"errors"
	"log"
	"net/http"

	"voltana-api/internal/middleware"
	"voltana-api/internal/repository"
	"voltana-api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CarHandler wires HTTP requests to CarService.
type CarHandler struct {
	cars *service.CarService
}

func NewCarHandler(cars *service.CarService) *CarHandler {
	return &CarHandler{cars: cars}
}

// name is omitempty (not required): when catalog_car_id is present the service
// defaults it to the catalog's name_fa; a blank name WITHOUT a catalog link is
// still rejected by the service-level validation.
type carRequest struct {
	Name          string         `json:"name"           binding:"omitempty,max=255"`
	EVModelID     *string        `json:"ev_model_id"    binding:"omitempty,uuid"`
	CatalogCarID  *string        `json:"catalog_car_id" binding:"omitempty,uuid"`
	SpecOverrides map[string]any `json:"spec_overrides"`
	LicensePlate  *string        `json:"license_plate"  binding:"omitempty,max=50"`
	OdometerKM    *int           `json:"odometer_km"    binding:"omitempty,min=0"`
}

func (req carRequest) toInput() repository.CarInput {
	in := repository.CarInput{Name: req.Name, LicensePlate: req.LicensePlate, SpecOverrides: req.SpecOverrides}
	if req.OdometerKM != nil {
		in.OdometerKM = *req.OdometerKM
	}
	if req.EVModelID != nil {
		// already validated as a UUID by binding
		id := uuid.MustParse(*req.EVModelID)
		in.EVModelID = &id
	}
	if req.CatalogCarID != nil {
		id := uuid.MustParse(*req.CatalogCarID)
		in.CatalogCarID = &id
	}
	return in
}

// POST /v1/cars
func (h *CarHandler) Create(c *gin.Context) {
	var req carRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apiError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	car, err := h.cars.Create(c.Request.Context(), userID(c), req.toInput())
	if err != nil {
		handleCarError(c, err)
		return
	}
	c.JSON(http.StatusCreated, car)
}

// GET /v1/cars
func (h *CarHandler) List(c *gin.Context) {
	limit, offset := parsePagination(c)
	items, total, err := h.cars.List(c.Request.Context(), userID(c), limit, offset)
	if err != nil {
		handleCarError(c, err)
		return
	}
	effLimit, effOffset := service.ClampPagination(limit, offset)
	c.JSON(http.StatusOK, listResponse{Items: items, Limit: effLimit, Offset: effOffset, Total: total})
}

// GET /v1/cars/:id
func (h *CarHandler) Get(c *gin.Context) {
	id, ok := parseUUIDParam(c)
	if !ok {
		return
	}
	car, err := h.cars.Get(c.Request.Context(), userID(c), id)
	if err != nil {
		handleCarError(c, err)
		return
	}
	c.JSON(http.StatusOK, car)
}

// PUT /v1/cars/:id
func (h *CarHandler) Update(c *gin.Context) {
	id, ok := parseUUIDParam(c)
	if !ok {
		return
	}
	var req carRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apiError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	car, err := h.cars.Update(c.Request.Context(), userID(c), id, req.toInput())
	if err != nil {
		handleCarError(c, err)
		return
	}
	c.JSON(http.StatusOK, car)
}

// DELETE /v1/cars/:id
func (h *CarHandler) Delete(c *gin.Context) {
	id, ok := parseUUIDParam(c)
	if !ok {
		return
	}
	if err := h.cars.Delete(c.Request.Context(), userID(c), id); err != nil {
		handleCarError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ── helpers ─────────────────────────────────────────────────────────────────

// userID extracts the authenticated user set by the Auth middleware.
func userID(c *gin.Context) uuid.UUID {
	return c.MustGet(middleware.UserIDKey).(uuid.UUID)
}

// parseUUIDParam reads :id, writing a 400 and returning ok=false if malformed.
func parseUUIDParam(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		apiError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid id")
		return uuid.Nil, false
	}
	return id, true
}

func handleCarError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrValidation):
		apiError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	case errors.Is(err, service.ErrInvalidOverride):
		apiError(c, http.StatusBadRequest, "INVALID_OVERRIDE_KEY", err.Error())
	case errors.Is(err, service.ErrInvalidEVModelRef):
		apiError(c, http.StatusUnprocessableEntity, "INVALID_EV_MODEL", "ev_model_id does not reference an existing model")
	case errors.Is(err, service.ErrInvalidCatalogCar):
		apiError(c, http.StatusUnprocessableEntity, "INVALID_CATALOG_CAR", "catalog_car_id does not reference a catalog car")
	case errors.Is(err, service.ErrCarNotFound):
		apiError(c, http.StatusNotFound, "NOT_FOUND", "car not found")
	default:
		log.Printf("car handler: %v", err)
		apiError(c, http.StatusInternalServerError, "INTERNAL", "internal error")
	}
}
