package handlers

import (
	"errors"
	"focuz-api/globals"
	"focuz-api/repository"
	"focuz-api/types"
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
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeValidation, "Invalid space ID"))
		return
	}
	userID := c.GetInt("userId")
	roleID, err := h.spacesRepo.GetUserRoleIDInSpace(userID, spaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}
	if roleID == 0 || roleID != globals.DefaultOwnerRoleID {
		c.JSON(http.StatusForbidden, types.NewErrorResponse(types.ErrorCodeForbidden, "No permission"))
		return
	}

	var req struct {
		Name        string   `json:"name" binding:"required"`
		ValueType   string   `json:"value_type" binding:"required"`
		Unit        *string  `json:"unit"`
		MinValue    *float64 `json:"min_value"`
		MaxValue    *float64 `json:"max_value"`
		Aggregation string   `json:"aggregation" binding:"required"`
		CategoryID  *int     `json:"category_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeValidation, err.Error()))
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
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeValidation, "Invalid value_type. Allowed: integer, float, text, boolean, time"))
		return
	}

	validAggregations := map[string]bool{
		"sum": true, "avg": true, "count": true, "min": true, "max": true,
		"and": true, "or": true,
		"count_true": true, "count_false": true,
		"percentage_true": true, "percentage_false": true,
	}
	if !validAggregations[strings.ToLower(req.Aggregation)] {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeValidation, "Invalid aggregation. Allowed: sum, avg, count, min, max, and, or, count_true, count_false, percentage_true, percentage_false"))
		return
	}
	if req.MinValue != nil && req.MaxValue != nil && *req.MinValue > *req.MaxValue {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeValidation, "min_value cannot be greater than max_value"))
		return
	}
	if req.ValueType == "text" && strings.ToLower(req.Aggregation) != "count" {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeValidation, "text supports only count aggregation"))
		return
	}
	boolAggSet := map[string]bool{
		"and": true, "or": true, "count_true": true, "count_false": true, "percentage_true": true, "percentage_false": true,
	}
	if req.ValueType == "boolean" && !boolAggSet[strings.ToLower(req.Aggregation)] {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeValidation, "invalid aggregation for boolean"))
		return
	}
	if req.ValueType == "time" && req.Unit != nil && *req.Unit != "" {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeValidation, "time cannot have a unit"))
		return
	}

	spacePtr := &spaceID
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
			c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeConflict, "type name already exists in this space"))
		} else {
			c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		}
		return
	}
	c.JSON(http.StatusCreated, types.NewSuccessResponse(created))
}

func (h *ActivityTypesHandler) DeleteActivityType(c *gin.Context) {
	spaceID, err := strconv.Atoi(c.Param("spaceId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeValidation, "Invalid space ID"))
		return
	}
	typeID, err := strconv.Atoi(c.Param("typeId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeValidation, "Invalid type ID"))
		return
	}
	userID := c.GetInt("userId")
	roleID, err := h.spacesRepo.GetUserRoleIDInSpace(userID, spaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}
	if roleID == 0 || roleID != globals.DefaultOwnerRoleID {
		c.JSON(http.StatusForbidden, types.NewErrorResponse(types.ErrorCodeForbidden, "No permission"))
		return
	}

	activityType, err := h.repo.GetActivityTypeByID(typeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}
	if activityType == nil || activityType.IsDeleted {
		c.JSON(http.StatusNotFound, types.NewErrorResponse(types.ErrorCodeNotFound, "Activity type not found"))
		return
	}
	if activityType.IsDefault {
		c.JSON(http.StatusForbidden, types.NewErrorResponse(types.ErrorCodeForbidden, "Cannot delete default activity type"))
		return
	}

	err = h.repo.UpdateActivityTypeDeleted(typeID, true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}
	c.JSON(http.StatusOK, types.NewSuccessResponse(gin.H{"message": "Activity type deleted successfully"}))
}

func (h *ActivityTypesHandler) RestoreActivityType(c *gin.Context) {
	spaceID, err := strconv.Atoi(c.Param("spaceId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeValidation, "Invalid space ID"))
		return
	}
	typeID, err := strconv.Atoi(c.Param("typeId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeValidation, "Invalid type ID"))
		return
	}
	userID := c.GetInt("userId")
	roleID, err := h.spacesRepo.GetUserRoleIDInSpace(userID, spaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}
	if roleID == 0 || roleID != globals.DefaultOwnerRoleID {
		c.JSON(http.StatusForbidden, types.NewErrorResponse(types.ErrorCodeForbidden, "No permission"))
		return
	}
	activityType, err := h.repo.GetActivityTypeByID(typeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}
	if activityType == nil {
		c.JSON(http.StatusNotFound, types.NewErrorResponse(types.ErrorCodeNotFound, "Activity type not found"))
		return
	}
	if activityType.IsDefault {
		c.JSON(http.StatusForbidden, types.NewErrorResponse(types.ErrorCodeForbidden, "Cannot restore default activity type"))
		return
	}
	err = h.repo.UpdateActivityTypeDeleted(typeID, false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}
	c.JSON(http.StatusOK, types.NewSuccessResponse(gin.H{"message": "Activity type restored successfully"}))
}

func (h *ActivityTypesHandler) GetActivityTypesBySpace(c *gin.Context) {
	spaceID, err := strconv.Atoi(c.Param("spaceId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeValidation, "Invalid space ID"))
		return
	}
	userID := c.GetInt("userId")
	roleID, err := h.spacesRepo.GetUserRoleIDInSpace(userID, spaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}
	if roleID == 0 {
		c.JSON(http.StatusForbidden, types.NewErrorResponse(types.ErrorCodeForbidden, "No access to the space"))
		return
	}
	activityTypes, err := h.repo.GetActivityTypesBySpace(spaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}
	c.JSON(http.StatusOK, types.NewSuccessResponse(activityTypes))
}
