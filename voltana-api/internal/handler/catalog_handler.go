package handler

import (
	"log"
	"net/http"

	"voltana-api/internal/service"

	"github.com/gin-gonic/gin"
)

// CatalogHandler serves the read-only rich EV catalog (TASK-0033).
type CatalogHandler struct {
	catalog *service.CatalogService
}

func NewCatalogHandler(catalog *service.CatalogService) *CatalogHandler {
	return &CatalogHandler{catalog: catalog}
}

// GET /v1/cars/catalog
func (h *CatalogHandler) List(c *gin.Context) {
	cars, err := h.catalog.List(c.Request.Context())
	if err != nil {
		log.Printf("catalog handler: %v", err)
		apiError(c, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}
	c.JSON(http.StatusOK, gin.H{"cars": cars})
}
