package handler

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"voltana-api/internal/domain"
	"voltana-api/internal/service"

	"github.com/gin-gonic/gin"
)

// StationHandler wires HTTP requests to StationService. Reads are open to any
// authed user; writes are gated by the AdminOnly middleware at the router.
type StationHandler struct {
	stations *service.StationService
}

func NewStationHandler(stations *service.StationService) *StationHandler {
	return &StationHandler{stations: stations}
}

type stationRequest struct {
	Name string `json:"name" binding:"required,max=255"`
	// Pointers so `required` distinguishes "omitted" (nil → 400) from a valid
	// latitude:0 / longitude:0 (the equator / prime meridian) — a plain float64
	// with `required` would reject 0 as the zero value. Bounds are validated in
	// the service so the error carries a descriptive message.
	Latitude       *float64 `json:"latitude"        binding:"required"`
	Longitude      *float64 `json:"longitude"       binding:"required"`
	Address        *string  `json:"address"         binding:"omitempty,max=500"`
	ConnectorTypes *string  `json:"connector_types" binding:"omitempty,max=255"`
	PowerKW        *int     `json:"power_kw"        binding:"omitempty,min=1"`
	Operator       *string  `json:"operator"        binding:"omitempty,max=255"`
}

func (req stationRequest) toInput() domain.StationInput {
	return domain.StationInput{
		Name:           req.Name,
		Latitude:       *req.Latitude, // non-nil guaranteed by binding:"required"
		Longitude:      *req.Longitude,
		Address:        req.Address,
		ConnectorTypes: req.ConnectorTypes,
		PowerKW:        req.PowerKW,
		Operator:       req.Operator,
	}
}

// GET /v1/stations  — markers; optional bounding-box filter
// (?min_lat&max_lat&min_lng&max_lng, all four together).
func (h *StationHandler) List(c *gin.Context) {
	bounds, ok := parseBounds(c)
	if !ok {
		return
	}
	items, err := h.stations.List(c.Request.Context(), bounds)
	if err != nil {
		handleStationError(c, err)
		return
	}
	// Bounded reference data returned in full; keep the standard list envelope
	// for consistency (limit == total, offset == 0).
	n := len(items)
	c.JSON(http.StatusOK, listResponse{Items: items, Limit: n, Offset: 0, Total: n})
}

// GET /v1/stations/:id
func (h *StationHandler) Get(c *gin.Context) {
	id, ok := parseUUIDParam(c)
	if !ok {
		return
	}
	st, err := h.stations.Get(c.Request.Context(), id)
	if err != nil {
		handleStationError(c, err)
		return
	}
	c.JSON(http.StatusOK, st)
}

// POST /v1/stations  (admin-only)
func (h *StationHandler) Create(c *gin.Context) {
	var req stationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apiError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	st, err := h.stations.Create(c.Request.Context(), req.toInput())
	if err != nil {
		handleStationError(c, err)
		return
	}
	c.JSON(http.StatusCreated, st)
}

// PUT /v1/stations/:id  (admin-only)
func (h *StationHandler) Update(c *gin.Context) {
	id, ok := parseUUIDParam(c)
	if !ok {
		return
	}
	var req stationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apiError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	st, err := h.stations.Update(c.Request.Context(), id, req.toInput())
	if err != nil {
		handleStationError(c, err)
		return
	}
	c.JSON(http.StatusOK, st)
}

// DELETE /v1/stations/:id  (admin-only)
func (h *StationHandler) Delete(c *gin.Context) {
	id, ok := parseUUIDParam(c)
	if !ok {
		return
	}
	if err := h.stations.Delete(c.Request.Context(), id); err != nil {
		handleStationError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ── helpers ─────────────────────────────────────────────────────────────────

// parseBounds reads the optional bbox query params. Returns (nil, true) when
// none are present (full set), (bounds, true) when all four parse, and writes a
// 400 + returns ok=false when the box is partial or malformed.
func parseBounds(c *gin.Context) (*domain.StationBounds, bool) {
	raw := map[string]string{
		"min_lat": c.Query("min_lat"),
		"max_lat": c.Query("max_lat"),
		"min_lng": c.Query("min_lng"),
		"max_lng": c.Query("max_lng"),
	}
	present := 0
	for _, v := range raw {
		if v != "" {
			present++
		}
	}
	if present == 0 {
		return nil, true
	}
	if present != 4 {
		apiError(c, http.StatusBadRequest, "INVALID_REQUEST",
			"bounding box requires all of min_lat, max_lat, min_lng, max_lng")
		return nil, false
	}
	b := &domain.StationBounds{}
	var err error
	if b.MinLat, err = strconv.ParseFloat(raw["min_lat"], 64); err != nil {
		return badBounds(c)
	}
	if b.MaxLat, err = strconv.ParseFloat(raw["max_lat"], 64); err != nil {
		return badBounds(c)
	}
	if b.MinLng, err = strconv.ParseFloat(raw["min_lng"], 64); err != nil {
		return badBounds(c)
	}
	if b.MaxLng, err = strconv.ParseFloat(raw["max_lng"], 64); err != nil {
		return badBounds(c)
	}
	return b, true
}

func badBounds(c *gin.Context) (*domain.StationBounds, bool) {
	apiError(c, http.StatusBadRequest, "INVALID_REQUEST", "bounding box values must be numbers")
	return nil, false
}

func handleStationError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrValidation):
		apiError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	case errors.Is(err, service.ErrStationNotFound):
		apiError(c, http.StatusNotFound, "NOT_FOUND", "station not found")
	default:
		log.Printf("station handler: %v", err)
		apiError(c, http.StatusInternalServerError, "INTERNAL", "internal error")
	}
}
