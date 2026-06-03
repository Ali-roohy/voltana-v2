package handler

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"voltana-api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AnalyticsHandler serves battery-health and recommendation endpoints.
type AnalyticsHandler struct {
	analytics *service.AnalyticsService
}

func NewAnalyticsHandler(analytics *service.AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{analytics: analytics}
}

// insufficientDataResponse is returned (with 200) when there is not yet enough
// qualifying charging history to estimate SOH — a normal state, not an error.
type insufficientDataResponse struct {
	Status             string `json:"status"`
	QualifyingSessions int    `json:"qualifying_sessions"`
}

// GET /v1/analytics/battery/:car_id
func (h *AnalyticsHandler) Battery(c *gin.Context) {
	carID, ok := parseCarIDParam(c)
	if !ok {
		return
	}
	res, err := h.analytics.GetBattery(c.Request.Context(), userID(c), carID)
	if err != nil {
		handleAnalyticsError(c, err)
		return
	}
	if res.Snapshot == nil {
		c.JSON(http.StatusOK, insufficientDataResponse{Status: "insufficient_data", QualifyingSessions: res.QualifyingSessions})
		return
	}
	c.JSON(http.StatusOK, res.Snapshot)
}

// GET /v1/analytics/recommendations/:car_id
func (h *AnalyticsHandler) Recommendations(c *gin.Context) {
	carID, ok := parseCarIDParam(c)
	if !ok {
		return
	}
	rec, err := h.analytics.GetRecommendations(c.Request.Context(), userID(c), carID)
	if err != nil {
		handleAnalyticsError(c, err)
		return
	}
	c.JSON(http.StatusOK, rec)
}

// GET /v1/analytics/dashboard
func (h *AnalyticsHandler) Dashboard(c *gin.Context) {
	stats, err := h.analytics.GetDashboard(c.Request.Context(), userID(c))
	if err != nil {
		handleAnalyticsError(c, err)
		return
	}
	c.JSON(http.StatusOK, stats)
}

// GET /v1/analytics/battery/:car_id/history?limit=30
func (h *AnalyticsHandler) BatteryHistory(c *gin.Context) {
	carID, ok := parseCarIDParam(c)
	if !ok {
		return
	}
	limit := 0
	if v, err := strconv.Atoi(c.Query("limit")); err == nil {
		limit = v // service applies default/clamp
	}
	items, err := h.analytics.GetBatteryHistory(c.Request.Context(), userID(c), carID, limit)
	if err != nil {
		handleAnalyticsError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

// ── helpers ─────────────────────────────────────────────────────────────────

// parseCarIDParam reads :car_id, writing a 400 and returning ok=false if malformed.
func parseCarIDParam(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("car_id"))
	if err != nil {
		apiError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid car_id")
		return uuid.Nil, false
	}
	return id, true
}

func handleAnalyticsError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrCarNotFound):
		apiError(c, http.StatusNotFound, "NOT_FOUND", "car not found")
	default:
		log.Printf("analytics handler: %v", err)
		apiError(c, http.StatusInternalServerError, "INTERNAL", "internal error")
	}
}
