package handlers

import (
	"focuz-api/globals"
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

	// We treat the new space creation as "owner" role, using the global default.
	space, err := h.spacesRepo.CreateSpace(req.Name, globals.DefaultOwnerRoleID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Invite the current user with the owner role.
	err = h.spacesRepo.InviteUserToSpace(userID, space.ID, globals.DefaultOwnerRoleID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, space)
}

func (h *SpacesHandler) UpdateSpace(c *gin.Context) {
	spaceID, err := strconv.Atoi(c.Param("spaceId"))
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
	userRoleID, err := h.spacesRepo.GetUserRoleIDInSpace(userID, spaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if userRoleID == 0 || userRoleID != globals.DefaultOwnerRoleID {
		c.JSON(http.StatusForbidden, gin.H{"error": "No permission to edit space"})
		return
	}

	err = h.spacesRepo.UpdateSpaceName(spaceID, req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *SpacesHandler) DeleteSpace(c *gin.Context) {
	spaceID, err := strconv.Atoi(c.Param("spaceId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid space ID"})
		return
	}
	userID := c.GetInt("userId")
	userRoleID, err := h.spacesRepo.GetUserRoleIDInSpace(userID, spaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if userRoleID == 0 || userRoleID != globals.DefaultOwnerRoleID {
		c.JSON(http.StatusForbidden, gin.H{"error": "No permission to delete space"})
		return
	}
	err = h.spacesRepo.SetSpaceDeleted(spaceID, true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *SpacesHandler) RestoreSpace(c *gin.Context) {
	spaceID, err := strconv.Atoi(c.Param("spaceId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid space ID"})
		return
	}
	userID := c.GetInt("userId")
	userRoleID, err := h.spacesRepo.GetUserRoleIDInSpace(userID, spaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if userRoleID == 0 || userRoleID != globals.DefaultOwnerRoleID {
		c.JSON(http.StatusForbidden, gin.H{"error": "No permission to restore space"})
		return
	}
	err = h.spacesRepo.SetSpaceDeleted(spaceID, false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *SpacesHandler) InviteUser(c *gin.Context) {
	spaceID, err := strconv.Atoi(c.Param("spaceId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid space ID"})
		return
	}
	userID := c.GetInt("userId")
	userRoleID, err := h.spacesRepo.GetUserRoleIDInSpace(userID, spaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if userRoleID == 0 || userRoleID != globals.DefaultOwnerRoleID {
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
	err = h.spacesRepo.InviteUserToSpace(req.UserID, spaceID, roleGuest.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *SpacesHandler) GetAccessibleSpaces(c *gin.Context) {
	userID := c.GetInt("userId")
	spaces, err := h.spacesRepo.GetSpacesForUser(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, spaces)
}

func (h *SpacesHandler) RemoveUser(c *gin.Context) {
	spaceID, err := strconv.Atoi(c.Param("spaceId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid space ID"})
		return
	}
	userID := c.GetInt("userId")
	userRoleID, err := h.spacesRepo.GetUserRoleIDInSpace(userID, spaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if userRoleID == 0 || userRoleID != globals.DefaultOwnerRoleID {
		c.JSON(http.StatusForbidden, gin.H{"error": "No permission to remove participants"})
		return
	}

	userToRemoveID, err := strconv.Atoi(c.Param("userId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}
	targetRoleID, err := h.spacesRepo.GetUserRoleIDInSpace(userToRemoveID, spaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if targetRoleID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Participant not found in space"})
		return
	}
	if targetRoleID == globals.DefaultOwnerRoleID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot remove the creator"})
		return
	}
	err = h.spacesRepo.RemoveUserFromSpace(userToRemoveID, spaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *SpacesHandler) GetUsersInSpace(c *gin.Context) {
	spaceID, err := strconv.Atoi(c.Param("spaceId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid space ID"})
		return
	}
	userID := c.GetInt("userId")
	userRoleID, err := h.spacesRepo.GetUserRoleIDInSpace(userID, spaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if userRoleID == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "No access to the space"})
		return
	}
	participants, err := h.spacesRepo.GetUsersInSpace(spaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, participants)
}
