package handler

import (
	"errors"
	"log"
	"net/http"

	"voltana-api/internal/service"

	"github.com/gin-gonic/gin"
)

// EVModelHandler serves the read-only EV catalog.
type EVModelHandler struct {
	models *service.EVModelService
}

func NewEVModelHandler(models *service.EVModelService) *EVModelHandler {
	return &EVModelHandler{models: models}
}

// GET /v1/ev-models
func (h *EVModelHandler) List(c *gin.Context) {
	limit, offset := parsePagination(c)
	items, total, err := h.models.List(c.Request.Context(), c.Query("q"), limit, offset)
	if err != nil {
		log.Printf("ev-model handler: %v", err)
		apiError(c, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}
	effLimit, effOffset := service.ClampPagination(limit, offset)
	c.JSON(http.StatusOK, listResponse{Items: items, Limit: effLimit, Offset: effOffset, Total: total})
}

// GET /v1/ev-models/:id
func (h *EVModelHandler) Get(c *gin.Context) {
	id, ok := parseUUIDParam(c)
	if !ok {
		return
	}
	m, err := h.models.Get(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrEVModelNotFound) {
			apiError(c, http.StatusNotFound, "NOT_FOUND", "ev model not found")
			return
		}
		log.Printf("ev-model handler: %v", err)
		apiError(c, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}
	c.JSON(http.StatusOK, m)
}
