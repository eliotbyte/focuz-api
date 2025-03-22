package handlers

import (
	"focuz-api/repository"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type SpacesHandler struct {
	spacesRepo *repository.SpacesRepository
	rolesRepo  *repository.RolesRepository
}

func NewSpacesHandler(spacesRepo *repository.SpacesRepository, rolesRepo *repository.RolesRepository) *SpacesHandler {
	return &SpacesHandler{spacesRepo: spacesRepo, rolesRepo: rolesRepo}
}

func (h *SpacesHandler) CreateSpace(c *gin.Context) {
	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID := c.GetInt("userId")
	space, err := h.spacesRepo.CreateSpace(req.Name, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, space)
}

func (h *SpacesHandler) UpdateSpace(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid space ID"})
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID := c.GetInt("userId")
	canEdit, err := h.spacesRepo.CanUserEditSpace(userID, id)
	if err != nil || !canEdit {
		c.JSON(http.StatusForbidden, gin.H{"error": "No permission to edit space"})
		return
	}
	err = h.spacesRepo.UpdateSpaceName(id, req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *SpacesHandler) DeleteSpace(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid space ID"})
		return
	}
	userID := c.GetInt("userId")
	canEdit, err := h.spacesRepo.CanUserEditSpace(userID, id)
	if err != nil || !canEdit {
		c.JSON(http.StatusForbidden, gin.H{"error": "No permission to delete space"})
		return
	}
	err = h.spacesRepo.SetSpaceDeleted(id, true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *SpacesHandler) RestoreSpace(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid space ID"})
		return
	}
	userID := c.GetInt("userId")
	canEdit, err := h.spacesRepo.CanUserEditSpace(userID, id)
	if err != nil || !canEdit {
		c.JSON(http.StatusForbidden, gin.H{"error": "No permission to restore space"})
		return
	}
	err = h.spacesRepo.SetSpaceDeleted(id, false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *SpacesHandler) InviteUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid space ID"})
		return
	}
	userID := c.GetInt("userId")
	canEdit, err := h.spacesRepo.CanUserEditSpace(userID, id)
	if err != nil || !canEdit {
		c.JSON(http.StatusForbidden, gin.H{"error": "No permission to invite user"})
		return
	}
	var req struct {
		UserID int `json:"userId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	roleGuest, err := h.rolesRepo.GetRoleByName("guest")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Role not found"})
		return
	}
	err = h.spacesRepo.InviteUserToSpace(req.UserID, id, roleGuest.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
