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
	c.JSON(http.StatusOK, types.ChartTypes)
}

func (h *ChartsHandler) GetPeriodTypes(c *gin.Context) {
	c.JSON(http.StatusOK, types.PeriodTypes)
}

func (h *ChartsHandler) CreateChart(c *gin.Context) {
	var req struct {
		TopicID        int `json:"topicId"`
		KindID         int `json:"kindId"`
		ActivityTypeID int `json:"activityTypeId"`
		PeriodID       int `json:"periodId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	chartType := types.GetChartTypeByID(req.KindID)
	if chartType == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chart kind"})
		return
	}

	periodType := types.GetPeriodTypeByID(req.PeriodID)
	if periodType == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid period"})
		return
	}

	userID := c.GetInt("userId")
	topic, err := h.topicsRepo.GetTopicByID(req.TopicID)
	if err != nil || topic == nil || topic.IsDeleted {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid topic"})
		return
	}

	topicType := types.GetTopicTypeByID(topic.TypeID)
	if topicType == nil || topicType.Name != "dashboard" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Charts can only be created in dashboard topics"})
		return
	}

	roleID, err := h.spacesRepo.GetUserRoleIDInSpace(userID, topic.SpaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if roleID == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "No access to the space"})
		return
	}

	activityType, err := h.activityTypesRepo.GetActivityTypeByID(req.ActivityTypeID)
	if err != nil || activityType == nil || activityType.IsDeleted {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid activity type"})
		return
	}

	chart, err := h.chartsRepo.CreateChart(userID, req.TopicID, req.KindID, req.ActivityTypeID, req.PeriodID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, chart)
}

func (h *ChartsHandler) DeleteChart(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	chart, err := h.chartsRepo.GetChartByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if chart == nil || chart.IsDeleted {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chart not found"})
		return
	}

	userID := c.GetInt("userId")
	topic, err := h.topicsRepo.GetTopicByID(chart.TopicID)
	if err != nil || topic == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Topic error"})
		return
	}

	roleID, err := h.spacesRepo.GetUserRoleIDInSpace(userID, topic.SpaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if roleID == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "No access to the space"})
		return
	}

	if err := h.chartsRepo.UpdateChartDeleted(id, true); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *ChartsHandler) RestoreChart(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	chart, err := h.chartsRepo.GetChartByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if chart == nil || !chart.IsDeleted {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chart not found"})
		return
	}

	userID := c.GetInt("userId")
	topic, err := h.topicsRepo.GetTopicByID(chart.TopicID)
	if err != nil || topic == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Topic error"})
		return
	}

	roleID, err := h.spacesRepo.GetUserRoleIDInSpace(userID, topic.SpaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if roleID == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "No access to the space"})
		return
	}

	if err := h.chartsRepo.UpdateChartDeleted(id, false); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *ChartsHandler) UpdateChart(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	chart, err := h.chartsRepo.GetChartByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if chart == nil || chart.IsDeleted {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chart not found"})
		return
	}

	var req struct {
		KindID         int `json:"kindId"`
		ActivityTypeID int `json:"activityTypeId"`
		PeriodID       int `json:"periodId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	chartType := types.GetChartTypeByID(req.KindID)
	if chartType == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chart kind"})
		return
	}

	periodType := types.GetPeriodTypeByID(req.PeriodID)
	if periodType == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid period"})
		return
	}

	userID := c.GetInt("userId")
	topic, err := h.topicsRepo.GetTopicByID(chart.TopicID)
	if err != nil || topic == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Topic error"})
		return
	}

	roleID, err := h.spacesRepo.GetUserRoleIDInSpace(userID, topic.SpaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if roleID == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "No access to the space"})
		return
	}

	activityType, err := h.activityTypesRepo.GetActivityTypeByID(req.ActivityTypeID)
	if err != nil || activityType == nil || activityType.IsDeleted {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid activity type"})
		return
	}

	if err := h.chartsRepo.UpdateChart(id, req.KindID, req.ActivityTypeID, req.PeriodID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	updatedChart, err := h.chartsRepo.GetChartByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, updatedChart)
}

func (h *ChartsHandler) GetCharts(c *gin.Context) {
	userID := c.GetInt("userId")
	spaceIDParam := c.Query("spaceId")
	if spaceIDParam == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "spaceId is required"})
		return
	}

	spaceID, err := strconv.Atoi(spaceIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid spaceId"})
		return
	}

	roleID, err := h.spacesRepo.GetUserRoleIDInSpace(userID, spaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if roleID == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "No access to the space"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"charts": charts,
		"total":  total,
	})
}
