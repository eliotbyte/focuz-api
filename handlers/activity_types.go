package handlers

import (
	"errors"
	"focuz-api/repository"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type ActivityTypesHandler struct {
	repo       *repository.ActivityTypesRepository
	spacesRepo *repository.SpacesRepository
}

func NewActivityTypesHandler(r *repository.ActivityTypesRepository, s *repository.SpacesRepository) *ActivityTypesHandler {
	return &ActivityTypesHandler{repo: r, spacesRepo: s}
}

func (h *ActivityTypesHandler) CreateActivityType(c *gin.Context) {
	spaceID, err := strconv.Atoi(c.Param("spaceId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid space ID"})
		return
	}
	userID := c.GetInt("userId")
	canEdit, err := h.spacesRepo.CanUserEditSpace(userID, spaceID)
	if err != nil || !canEdit {
		c.JSON(http.StatusForbidden, gin.H{"error": "No permission"})
		return
	}
	var req struct {
		Name        string   `json:"name"`
		ValueType   string   `json:"value_type"`
		Unit        *string  `json:"unit"`
		MinValue    *float64 `json:"min_value"`
		MaxValue    *float64 `json:"max_value"`
		Aggregation string   `json:"aggregation"`
		CategoryID  *int     `json:"category_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	validValueTypes := map[string]bool{
		"integer": true,
		"float":   true,
		"text":    true,
		"boolean": true,
		"time":    true,
	}
	if !validValueTypes[req.ValueType] {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid value_type. Allowed: integer, float, text, boolean, time",
		})
		return
	}

	validAggregations := map[string]bool{
		"sum": true, "avg": true, "count": true, "min": true, "max": true,
		"and": true, "or": true,
		"count_true": true, "count_false": true,
		"percentage_true": true, "percentage_false": true,
	}
	if !validAggregations[strings.ToLower(req.Aggregation)] {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid aggregation. Allowed: sum, avg, count, min, max, and, or, count_true, count_false, percentage_true, percentage_false",
		})
		return
	}
	if req.MinValue != nil && req.MaxValue != nil && *req.MinValue > *req.MaxValue {
		c.JSON(http.StatusBadRequest, gin.H{"error": "min_value cannot be greater than max_value"})
		return
	}
	if req.ValueType == "text" && strings.ToLower(req.Aggregation) != "count" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "text supports only count aggregation"})
		return
	}
	boolAggSet := map[string]bool{
		"and": true, "or": true, "count_true": true, "count_false": true, "percentage_true": true, "percentage_false": true,
	}
	if req.ValueType == "boolean" && !boolAggSet[strings.ToLower(req.Aggregation)] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid aggregation for boolean"})
		return
	}
	if req.ValueType == "time" && req.Unit != nil && *req.Unit != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "time cannot have a unit"})
		return
	}
	var spacePtr *int
	spacePtr = &spaceID
	created, err := h.repo.CreateActivityType(
		req.Name,
		req.ValueType,
		req.MinValue,
		req.MaxValue,
		req.Aggregation,
		spacePtr,
		req.CategoryID,
		req.Unit,
	)
	if err != nil {
		if errors.Is(err, errors.New("name conflict in this space")) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "type name already exists in this space"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusCreated, created)
}

func (h *ActivityTypesHandler) DeleteActivityType(c *gin.Context) {
	spaceID, err := strconv.Atoi(c.Param("spaceId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid space ID"})
		return
	}
	typeID, err := strconv.Atoi(c.Param("typeId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid type ID"})
		return
	}
	userID := c.GetInt("userId")
	canEdit, err := h.spacesRepo.CanUserEditSpace(userID, spaceID)
	if err != nil || !canEdit {
		c.JSON(http.StatusForbidden, gin.H{"error": "No permission"})
		return
	}
	err = h.repo.UpdateActivityTypeDeleted(typeID, true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *ActivityTypesHandler) RestoreActivityType(c *gin.Context) {
	spaceID, err := strconv.Atoi(c.Param("spaceId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid space ID"})
		return
	}
	typeID, err := strconv.Atoi(c.Param("typeId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid type ID"})
		return
	}
	userID := c.GetInt("userId")
	canEdit, err := h.spacesRepo.CanUserEditSpace(userID, spaceID)
	if err != nil || !canEdit {
		c.JSON(http.StatusForbidden, gin.H{"error": "No permission"})
		return
	}
	err = h.repo.UpdateActivityTypeDeleted(typeID, false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// New method
func (h *ActivityTypesHandler) GetActivityTypesBySpace(c *gin.Context) {
	spaceID, err := strconv.Atoi(c.Param("spaceId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid space ID"})
		return
	}
	userID := c.GetInt("userId")
	hasAccess, _, err := h.spacesRepo.UserHasAccessToSpace(userID, spaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !hasAccess {
		c.JSON(http.StatusForbidden, gin.H{"error": "No access to the space"})
		return
	}
	activityTypes, err := h.repo.GetActivityTypesBySpace(spaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, activityTypes)
}
