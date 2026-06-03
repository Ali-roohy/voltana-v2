package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

// apiError writes the standard error envelope: {"error","code"}.
func apiError(c *gin.Context, status int, code, msg string) {
	c.JSON(status, gin.H{"error": msg, "code": code})
}

// listResponse is the standard collection envelope.
type listResponse struct {
	Items  any `json:"items"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Total  int `json:"total"`
}

// parsePagination reads ?limit and ?offset. Missing/invalid values fall back to
// 0 here; the service layer applies defaults and clamps the range.
func parsePagination(c *gin.Context) (limit, offset int) {
	if v, err := strconv.Atoi(c.Query("limit")); err == nil {
		limit = v
	}
	if v, err := strconv.Atoi(c.Query("offset")); err == nil {
		offset = v
	}
	return limit, offset
}
