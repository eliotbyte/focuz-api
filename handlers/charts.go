package handlers

import (
	"focuz-api/models"
	"focuz-api/repository"
	"focuz-api/types"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ChartsHandler struct {
	chartsRepo        *repository.ChartsRepository
	spacesRepo        *repository.SpacesRepository
	topicsRepo        *repository.TopicsRepository
	activityTypesRepo *repository.ActivityTypesRepository
}

func NewChartsHandler(
	cr *repository.ChartsRepository,
	sr *repository.SpacesRepository,
	tr *repository.TopicsRepository,
	atr *repository.ActivityTypesRepository,
) *ChartsHandler {
	return &ChartsHandler{
		chartsRepo:        cr,
		spacesRepo:        sr,
		topicsRepo:        tr,
		activityTypesRepo: atr,
	}
}

func (h *ChartsHandler) GetChartTypes(c *gin.Context) {
	c.JSON(http.StatusOK, types.NewSuccessResponse(types.ChartTypes))
}

func (h *ChartsHandler) GetPeriodTypes(c *gin.Context) {
	c.JSON(http.StatusOK, types.NewSuccessResponse(types.PeriodTypes))
}

func (h *ChartsHandler) CreateChart(c *gin.Context) {
	var req struct {
		TopicID        int `json:"topicId"`
		KindID         int `json:"kindId"`
		ActivityTypeID int `json:"activityTypeId"`
		PeriodID       int `json:"periodId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeValidation, err.Error()))
		return
	}

	chartType := types.GetChartTypeByID(req.KindID)
	if chartType == nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeInvalidRequest, "Invalid chart kind"))
		return
	}

	periodType := types.GetPeriodTypeByID(req.PeriodID)
	if periodType == nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeInvalidRequest, "Invalid period"))
		return
	}

	userID := c.GetInt("userId")
	topic, err := h.topicsRepo.GetTopicByID(req.TopicID)
	if err != nil || topic == nil || topic.IsDeleted {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeInvalidRequest, "Invalid topic"))
		return
	}

	topicType := types.GetTopicTypeByID(topic.TypeID)
	if topicType == nil || topicType.Name != "dashboard" {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeInvalidRequest, "Charts can only be created in dashboard topics"))
		return
	}

	roleID, err := h.spacesRepo.GetUserRoleIDInSpace(userID, topic.SpaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}
	if roleID == 0 {
		c.JSON(http.StatusForbidden, types.NewErrorResponse(types.ErrorCodeForbidden, "No access to the space"))
		return
	}

	activityType, err := h.activityTypesRepo.GetActivityTypeByID(req.ActivityTypeID)
	if err != nil || activityType == nil || activityType.IsDeleted {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeInvalidRequest, "Invalid activity type"))
		return
	}

	chart, err := h.chartsRepo.CreateChart(userID, req.TopicID, req.KindID, req.ActivityTypeID, req.PeriodID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}

	c.JSON(http.StatusCreated, types.NewSuccessResponse(chart))
}

func (h *ChartsHandler) DeleteChart(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeValidation, "Invalid ID"))
		return
	}

	chart, err := h.chartsRepo.GetChartByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}
	if chart == nil || chart.IsDeleted {
		c.JSON(http.StatusNotFound, types.NewErrorResponse(types.ErrorCodeNotFound, "Chart not found"))
		return
	}

	userID := c.GetInt("userId")
	topic, err := h.topicsRepo.GetTopicByID(chart.TopicID)
	if err != nil || topic == nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeInvalidRequest, "Topic error"))
		return
	}

	roleID, err := h.spacesRepo.GetUserRoleIDInSpace(userID, topic.SpaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}
	if roleID == 0 {
		c.JSON(http.StatusForbidden, types.NewErrorResponse(types.ErrorCodeForbidden, "No access to the space"))
		return
	}

	if err := h.chartsRepo.UpdateChartDeleted(id, true); err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}

	c.JSON(http.StatusOK, types.NewSuccessResponse(gin.H{"message": "Chart deleted successfully"}))
}

func (h *ChartsHandler) RestoreChart(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeValidation, "Invalid ID"))
		return
	}

	chart, err := h.chartsRepo.GetChartByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}
	if chart == nil || !chart.IsDeleted {
		c.JSON(http.StatusNotFound, types.NewErrorResponse(types.ErrorCodeNotFound, "Chart not found"))
		return
	}

	userID := c.GetInt("userId")
	topic, err := h.topicsRepo.GetTopicByID(chart.TopicID)
	if err != nil || topic == nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeInvalidRequest, "Topic error"))
		return
	}

	roleID, err := h.spacesRepo.GetUserRoleIDInSpace(userID, topic.SpaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}
	if roleID == 0 {
		c.JSON(http.StatusForbidden, types.NewErrorResponse(types.ErrorCodeForbidden, "No access to the space"))
		return
	}

	if err := h.chartsRepo.UpdateChartDeleted(id, false); err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}

	c.JSON(http.StatusOK, types.NewSuccessResponse(gin.H{"message": "Chart restored successfully"}))
}

func (h *ChartsHandler) UpdateChart(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeValidation, "Invalid ID"))
		return
	}

	chart, err := h.chartsRepo.GetChartByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}
	if chart == nil || chart.IsDeleted {
		c.JSON(http.StatusNotFound, types.NewErrorResponse(types.ErrorCodeNotFound, "Chart not found"))
		return
	}

	userID := c.GetInt("userId")
	topic, err := h.topicsRepo.GetTopicByID(chart.TopicID)
	if err != nil || topic == nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeInvalidRequest, "Topic error"))
		return
	}

	roleID, err := h.spacesRepo.GetUserRoleIDInSpace(userID, topic.SpaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}
	if roleID == 0 {
		c.JSON(http.StatusForbidden, types.NewErrorResponse(types.ErrorCodeForbidden, "No access to the space"))
		return
	}

	var req struct {
		KindID         *int `json:"kindId"`
		ActivityTypeID *int `json:"activityTypeId"`
		PeriodID       *int `json:"periodId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeValidation, err.Error()))
		return
	}

	if req.KindID != nil {
		chartType := types.GetChartTypeByID(*req.KindID)
		if chartType == nil {
			c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeInvalidRequest, "Invalid chart kind"))
			return
		}
	}

	if req.PeriodID != nil {
		periodType := types.GetPeriodTypeByID(*req.PeriodID)
		if periodType == nil {
			c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeInvalidRequest, "Invalid period"))
			return
		}
	}

	if req.ActivityTypeID != nil {
		activityType, err := h.activityTypesRepo.GetActivityTypeByID(*req.ActivityTypeID)
		if err != nil || activityType == nil || activityType.IsDeleted {
			c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeInvalidRequest, "Invalid activity type"))
			return
		}
	}

	// Get current values and update only provided fields
	kindID := chart.KindID
	if req.KindID != nil {
		kindID = *req.KindID
	}
	activityTypeID := chart.ActivityTypeID
	if req.ActivityTypeID != nil {
		activityTypeID = *req.ActivityTypeID
	}
	periodID := chart.PeriodID
	if req.PeriodID != nil {
		periodID = *req.PeriodID
	}

	err = h.chartsRepo.UpdateChart(id, kindID, activityTypeID, periodID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}

	c.JSON(http.StatusOK, types.NewSuccessResponse(gin.H{"message": "Chart updated successfully"}))
}

func (h *ChartsHandler) GetCharts(c *gin.Context) {
	userID := c.GetInt("userId")
	spaceIDParam := c.Query("spaceId")
	if spaceIDParam == "" {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeValidation, "spaceId is required"))
		return
	}

	spaceID, err := strconv.Atoi(spaceIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeValidation, "Invalid spaceId"))
		return
	}

	roleID, err := h.spacesRepo.GetUserRoleIDInSpace(userID, spaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}
	if roleID == 0 {
		c.JSON(http.StatusForbidden, types.NewErrorResponse(types.ErrorCodeForbidden, "No access to the space"))
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))

	topicIDParam := c.Query("topicId")
	var topicID *int
	if topicIDParam != "" {
		tmp, err := strconv.Atoi(topicIDParam)
		if err == nil {
			topicID = &tmp
		}
	}

	filters := models.ChartFilters{
		Page:     page,
		PageSize: pageSize,
		TopicID:  topicID,
	}

	charts, total, err := h.chartsRepo.GetCharts(spaceID, topicID, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}

	c.JSON(http.StatusOK, types.NewSuccessResponse(gin.H{
		"charts": charts,
		"total":  total,
	}))
}
