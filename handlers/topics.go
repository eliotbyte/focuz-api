package handlers

import (
	"focuz-api/globals"
	"focuz-api/repository"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type TopicsHandler struct {
	topicsRepo *repository.TopicsRepository
	spacesRepo *repository.SpacesRepository
}

func NewTopicsHandler(topicsRepo *repository.TopicsRepository, spacesRepo *repository.SpacesRepository) *TopicsHandler {
	return &TopicsHandler{topicsRepo: topicsRepo, spacesRepo: spacesRepo}
}

func (h *TopicsHandler) CreateTopic(c *gin.Context) {
	var req struct {
		SpaceID int    `json:"spaceId"`
		Name    string `json:"name"`
		TypeID  int    `json:"typeId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID := c.GetInt("userId")
	roleID, err := h.spacesRepo.GetUserRoleIDInSpace(userID, req.SpaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if roleID == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "No permission to create topic"})
		return
	}
	// Only owner can create a topic
	if roleID != globals.DefaultOwnerRoleID {
		c.JSON(http.StatusForbidden, gin.H{"error": "No permission to create topic"})
		return
	}
	topic, err := h.topicsRepo.CreateTopic(req.SpaceID, req.Name, req.TypeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, topic)
}

func (h *TopicsHandler) UpdateTopic(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid topic ID"})
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	topic, err := h.topicsRepo.GetTopicByID(id)
	if err != nil || topic == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Topic not found"})
		return
	}
	userID := c.GetInt("userId")
	roleID, err := h.spacesRepo.GetUserRoleIDInSpace(userID, topic.SpaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if roleID == 0 || roleID != globals.DefaultOwnerRoleID {
		c.JSON(http.StatusForbidden, gin.H{"error": "No permission to update topic"})
		return
	}
	err = h.topicsRepo.UpdateTopicName(id, req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *TopicsHandler) DeleteTopic(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid topic ID"})
		return
	}
	topic, err := h.topicsRepo.GetTopicByID(id)
	if err != nil || topic == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Topic not found"})
		return
	}
	userID := c.GetInt("userId")
	roleID, err := h.spacesRepo.GetUserRoleIDInSpace(userID, topic.SpaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if roleID == 0 || roleID != globals.DefaultOwnerRoleID {
		c.JSON(http.StatusForbidden, gin.H{"error": "No permission to delete topic"})
		return
	}
	err = h.topicsRepo.SetTopicDeleted(id, true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *TopicsHandler) RestoreTopic(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid topic ID"})
		return
	}
	topic, err := h.topicsRepo.GetTopicByID(id)
	if err != nil || topic == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Topic not found"})
		return
	}
	userID := c.GetInt("userId")
	roleID, err := h.spacesRepo.GetUserRoleIDInSpace(userID, topic.SpaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if roleID == 0 || roleID != globals.DefaultOwnerRoleID {
		c.JSON(http.StatusForbidden, gin.H{"error": "No permission to restore topic"})
		return
	}
	err = h.topicsRepo.SetTopicDeleted(id, false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *TopicsHandler) GetTopicsBySpace(c *gin.Context) {
	spaceID, err := strconv.Atoi(c.Param("spaceId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid space ID"})
		return
	}
	userID := c.GetInt("userId")
	roleID, err := h.spacesRepo.GetUserRoleIDInSpace(userID, spaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if roleID == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "No access to the space"})
		return
	}
	topics, err := h.topicsRepo.GetTopicsBySpace(spaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, topics)
}
