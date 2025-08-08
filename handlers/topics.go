package handlers

import (
	"focuz-api/globals"
	"focuz-api/repository"
	"focuz-api/types"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type TopicsHandler struct {
	topicsRepo *repository.TopicsRepository
	spacesRepo *repository.SpacesRepository
	rolesRepo  *repository.RolesRepository
}

func NewTopicsHandler(topicsRepo *repository.TopicsRepository, spacesRepo *repository.SpacesRepository, rolesRepo *repository.RolesRepository) *TopicsHandler {
	return &TopicsHandler{topicsRepo: topicsRepo, spacesRepo: spacesRepo, rolesRepo: rolesRepo}
}

func (h *TopicsHandler) GetTopicTypes(c *gin.Context) {
	c.JSON(http.StatusOK, types.NewSuccessResponse(types.TopicTypes))
}

func (h *TopicsHandler) CreateTopic(c *gin.Context) {
	var req struct {
		SpaceID int    `json:"spaceId" binding:"required"`
		Name    string `json:"name" binding:"required"`
		TypeID  int    `json:"typeId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeValidation, err.Error()))
		return
	}

	topicType := types.GetTopicTypeByID(req.TypeID)
	if topicType == nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeInvalidRequest, "Invalid topic type"))
		return
	}

	userID := c.GetInt("userId")
	roleID, err := h.spacesRepo.GetUserRoleIDInSpace(userID, req.SpaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}

	if roleID == 0 {
		c.JSON(http.StatusForbidden, types.NewErrorResponse(types.ErrorCodeForbidden, "No permission to create topic"))
		return
	}

	// Check if user has owner role by comparing roleID with DefaultOwnerRoleID
	if roleID != globals.DefaultOwnerRoleID {
		c.JSON(http.StatusForbidden, types.NewErrorResponse(types.ErrorCodeForbidden, "No permission to create topic"))
		return
	}

	topic, err := h.topicsRepo.CreateTopic(req.SpaceID, req.Name, req.TypeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}
	c.JSON(http.StatusCreated, types.NewSuccessResponse(topic))
}

func (h *TopicsHandler) UpdateTopic(c *gin.Context) {
	topicID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeValidation, "Invalid topic ID"))
		return
	}

	var req struct {
		Name   *string `json:"name"`
		TypeID *int    `json:"typeId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeValidation, err.Error()))
		return
	}

	topic, err := h.topicsRepo.GetTopicByID(topicID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}
	if topic == nil {
		c.JSON(http.StatusNotFound, types.NewErrorResponse(types.ErrorCodeNotFound, "Topic not found"))
		return
	}

	userID := c.GetInt("userId")
	roleID, err := h.spacesRepo.GetUserRoleIDInSpace(userID, topic.SpaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}
	if roleID == 0 {
		c.JSON(http.StatusForbidden, types.NewErrorResponse(types.ErrorCodeForbidden, "No permission to update topic"))
		return
	}

	// Check if user has owner role by comparing roleID with DefaultOwnerRoleID
	if roleID != globals.DefaultOwnerRoleID {
		c.JSON(http.StatusForbidden, types.NewErrorResponse(types.ErrorCodeForbidden, "No permission to update topic"))
		return
	}

	if req.TypeID != nil {
		topicType := types.GetTopicTypeByID(*req.TypeID)
		if topicType == nil {
			c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeInvalidRequest, "Invalid topic type"))
			return
		}
		// Note: TypeID update is not implemented in repository yet
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeInvalidRequest, "TypeID update not supported"))
		return
	}

	if req.Name != nil {
		err = h.topicsRepo.UpdateTopicName(topicID, *req.Name)
		if err != nil {
			c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
			return
		}
	}

	// Get updated topic
	updatedTopic, err := h.topicsRepo.GetTopicByID(topicID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}
	c.JSON(http.StatusOK, types.NewSuccessResponse(updatedTopic))
}

func (h *TopicsHandler) DeleteTopic(c *gin.Context) {
	topicID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeValidation, "Invalid topic ID"))
		return
	}

	topic, err := h.topicsRepo.GetTopicByID(topicID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}
	if topic == nil {
		c.JSON(http.StatusNotFound, types.NewErrorResponse(types.ErrorCodeNotFound, "Topic not found"))
		return
	}

	userID := c.GetInt("userId")
	roleID, err := h.spacesRepo.GetUserRoleIDInSpace(userID, topic.SpaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}
	if roleID == 0 {
		c.JSON(http.StatusForbidden, types.NewErrorResponse(types.ErrorCodeForbidden, "No permission to delete topic"))
		return
	}

	// Check if user has owner role by comparing roleID with DefaultOwnerRoleID
	if roleID != globals.DefaultOwnerRoleID {
		c.JSON(http.StatusForbidden, types.NewErrorResponse(types.ErrorCodeForbidden, "No permission to delete topic"))
		return
	}

	err = h.topicsRepo.SetTopicDeleted(topicID, true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}
	c.JSON(http.StatusOK, types.NewSuccessResponse(gin.H{"message": "Topic deleted successfully"}))
}

func (h *TopicsHandler) RestoreTopic(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeValidation, "Invalid topic ID"))
		return
	}
	topic, err := h.topicsRepo.GetTopicByID(id)
	if err != nil || topic == nil {
		c.JSON(http.StatusNotFound, types.NewErrorResponse(types.ErrorCodeNotFound, "Topic not found"))
		return
	}
	userID := c.GetInt("userId")
	roleID, err := h.spacesRepo.GetUserRoleIDInSpace(userID, topic.SpaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}
	if roleID == 0 || roleID != globals.DefaultOwnerRoleID {
		c.JSON(http.StatusForbidden, types.NewErrorResponse(types.ErrorCodeForbidden, "No permission to restore topic"))
		return
	}
	err = h.topicsRepo.SetTopicDeleted(id, false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}
	c.JSON(http.StatusOK, types.NewSuccessResponse(gin.H{"message": "Topic restored successfully"}))
}

func (h *TopicsHandler) GetTopicsBySpace(c *gin.Context) {
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

	// Use standardized pagination
	pagination := types.ParsePaginationParams(c)

	topics, total, err := h.topicsRepo.GetTopicsBySpacePaginated(spaceID, pagination.Offset, pagination.PageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}

	// Use standardized response with pagination
	response := pagination.BuildResponse(topics, total)
	c.JSON(http.StatusOK, types.NewSuccessResponse(response))
}
