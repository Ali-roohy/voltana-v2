package handler

import (
	"errors"
	"log"
	"net/http"
	"time"

	"voltana-api/internal/domain"
	"voltana-api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ChargingHandler wires HTTP requests to ChargingService.
type ChargingHandler struct {
	sessions *service.ChargingService
}

func NewChargingHandler(sessions *service.ChargingService) *ChargingHandler {
	return &ChargingHandler{sessions: sessions}
}

type chargingRequest struct {
	CarID            string     `json:"car_id"             binding:"required,uuid"`
	StartedAt        time.Time  `json:"started_at"         binding:"required"`
	EndedAt          *time.Time `json:"ended_at"`
	Location         *string    `json:"location"           binding:"omitempty,max=255"`
	KWhCharged       *float64   `json:"kwh_charged"        binding:"omitempty,gte=0"`
	EnergyPeakKWh    *float64   `json:"energy_peak_kwh"    binding:"omitempty,gte=0"`
	EnergyMidKWh     *float64   `json:"energy_mid_kwh"     binding:"omitempty,gte=0"`
	EnergyOffpeakKWh *float64   `json:"energy_offpeak_kwh" binding:"omitempty,gte=0"`
	StartSOC         *int       `json:"start_soc"          binding:"omitempty,min=0,max=100"`
	EndSOC           *int       `json:"end_soc"            binding:"omitempty,min=0,max=100"`
	Cost             *float64   `json:"cost"               binding:"omitempty,gte=0"`
	Notes            *string    `json:"notes"`
	OdometerKM       *int       `json:"odometer_km"        binding:"omitempty,min=0"`
	ChargePowerKW    *float64   `json:"charge_power_kw"    binding:"omitempty,gt=0"`
}

func (req chargingRequest) toInput() domain.ChargingInput {
	return domain.ChargingInput{
		CarID:            uuid.MustParse(req.CarID), // validated as a UUID by binding
		StartedAt:        req.StartedAt,
		EndedAt:          req.EndedAt,
		Location:         req.Location,
		KWhCharged:       req.KWhCharged,
		EnergyPeakKWh:    req.EnergyPeakKWh,
		EnergyMidKWh:     req.EnergyMidKWh,
		EnergyOffpeakKWh: req.EnergyOffpeakKWh,
		StartSOC:         req.StartSOC,
		EndSOC:           req.EndSOC,
		Cost:             req.Cost,
		Notes:            req.Notes,
		OdometerKM:       req.OdometerKM,
		ChargePowerKW:    req.ChargePowerKW,
	}
}

// POST /v1/charging-sessions
func (h *ChargingHandler) Create(c *gin.Context) {
	var req chargingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apiError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	sess, err := h.sessions.Create(c.Request.Context(), userID(c), req.toInput())
	if err != nil {
		handleChargingError(c, err)
		return
	}
	c.JSON(http.StatusCreated, sess)
}

// GET /v1/charging-sessions
func (h *ChargingHandler) List(c *gin.Context) {
	filter, ok := parseChargingFilter(c)
	if !ok {
		return
	}
	limit, offset := parsePagination(c)
	items, total, err := h.sessions.List(c.Request.Context(), userID(c), filter, limit, offset)
	if err != nil {
		handleChargingError(c, err)
		return
	}
	effLimit, effOffset := service.ClampPagination(limit, offset)
	c.JSON(http.StatusOK, listResponse{Items: items, Limit: effLimit, Offset: effOffset, Total: total})
}

// GET /v1/charging-sessions/:id
func (h *ChargingHandler) Get(c *gin.Context) {
	id, ok := parseUUIDParam(c)
	if !ok {
		return
	}
	sess, err := h.sessions.Get(c.Request.Context(), userID(c), id)
	if err != nil {
		handleChargingError(c, err)
		return
	}
	c.JSON(http.StatusOK, sess)
}

// PUT /v1/charging-sessions/:id
func (h *ChargingHandler) Update(c *gin.Context) {
	id, ok := parseUUIDParam(c)
	if !ok {
		return
	}
	var req chargingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apiError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	sess, err := h.sessions.Update(c.Request.Context(), userID(c), id, req.toInput())
	if err != nil {
		handleChargingError(c, err)
		return
	}
	c.JSON(http.StatusOK, sess)
}

// DELETE /v1/charging-sessions/:id
func (h *ChargingHandler) Delete(c *gin.Context) {
	id, ok := parseUUIDParam(c)
	if !ok {
		return
	}
	if err := h.sessions.Delete(c.Request.Context(), userID(c), id); err != nil {
		handleChargingError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ── helpers ─────────────────────────────────────────────────────────────────

// parseChargingFilter reads ?car_id, ?from, ?to. Malformed values write a 400
// and return ok=false.
func parseChargingFilter(c *gin.Context) (domain.ChargingFilter, bool) {
	var f domain.ChargingFilter
	if v := c.Query("car_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			apiError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid car_id")
			return f, false
		}
		f.CarID = &id
	}
	if v := c.Query("from"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			apiError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid from (want RFC3339)")
			return f, false
		}
		f.From = &t
	}
	if v := c.Query("to"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			apiError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid to (want RFC3339)")
			return f, false
		}
		f.To = &t
	}
	return f, true
}

func handleChargingError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrValidation):
		apiError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	case errors.Is(err, service.ErrInvalidCarRef):
		apiError(c, http.StatusUnprocessableEntity, "INVALID_CAR", "car_id does not reference one of your cars")
	case errors.Is(err, service.ErrOdometerNotIncreasing):
		apiError(c, http.StatusUnprocessableEntity, "ODOMETER_NOT_INCREASING", err.Error())
	case errors.Is(err, service.ErrChargingNotFound):
		apiError(c, http.StatusNotFound, "NOT_FOUND", "charging session not found")
	default:
		log.Printf("charging handler: %v", err)
		apiError(c, http.StatusInternalServerError, "INTERNAL", "internal error")
	}
}
