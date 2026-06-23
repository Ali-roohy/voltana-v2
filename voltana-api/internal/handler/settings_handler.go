package handler

import (
	"errors"
	"log"
	"net/http"

	"voltana-api/internal/domain"
	"voltana-api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// SettingsHandler wires HTTP requests to SettingsService.
type SettingsHandler struct {
	settings *service.SettingsService
}

func NewSettingsHandler(settings *service.SettingsService) *SettingsHandler {
	return &SettingsHandler{settings: settings}
}

// settingsRequest is the PUT body. PUT is a full replace: omitted rates default
// to 0; omitted/null default_car_id clears it.
type settingsRequest struct {
	DefaultCarID *string `json:"default_car_id" binding:"omitempty,uuid"`
	PeakRate     float64  `json:"peak_rate"      binding:"gte=0"`
	MidRate      float64  `json:"mid_rate"       binding:"gte=0"`
	OffpeakRate  float64  `json:"offpeak_rate"   binding:"gte=0"`
	Currency     string   `json:"currency"       binding:"omitempty,oneof=toman rial usd"`
	City         *string  `json:"city"           binding:"omitempty,max=120"`         // FEAT-2
	RegenFactor  *float64 `json:"regen_factor"   binding:"omitempty,gte=0,lte=1"`     // FEAT-4
}

func (req settingsRequest) toInput() domain.SettingsInput {
	in := domain.SettingsInput{
		PeakRate:    req.PeakRate,
		MidRate:     req.MidRate,
		OffpeakRate: req.OffpeakRate,
		Currency:    req.Currency,
		City:        req.City,
		RegenFactor: 0.10, // default when omitted (full-replace PUT)
	}
	if req.RegenFactor != nil {
		in.RegenFactor = *req.RegenFactor
	}
	if req.DefaultCarID != nil {
		id := uuid.MustParse(*req.DefaultCarID) // validated as a UUID by binding
		in.DefaultCarID = &id
	}
	return in
}

// GET /v1/settings
func (h *SettingsHandler) Get(c *gin.Context) {
	st, err := h.settings.Get(c.Request.Context(), userID(c))
	if err != nil {
		handleSettingsError(c, err)
		return
	}
	c.JSON(http.StatusOK, st)
}

// PUT /v1/settings
func (h *SettingsHandler) Update(c *gin.Context) {
	var req settingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apiError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	st, err := h.settings.Update(c.Request.Context(), userID(c), req.toInput())
	if err != nil {
		handleSettingsError(c, err)
		return
	}
	c.JSON(http.StatusOK, st)
}

func handleSettingsError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrValidation):
		apiError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	case errors.Is(err, service.ErrInvalidCarRef):
		apiError(c, http.StatusUnprocessableEntity, "INVALID_CAR", "default_car_id does not reference one of your cars")
	default:
		log.Printf("settings handler: %v", err)
		apiError(c, http.StatusInternalServerError, "INTERNAL", "internal error")
	}
}
