package handlers

import (
	"focuz-api/globals"
	"focuz-api/repository"
	"focuz-api/types"
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
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeValidation, err.Error()))
		return
	}
	userID := c.GetInt("userId")

	// Create the space with the current user as owner
	space, err := h.spacesRepo.CreateSpace(req.Name, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}

	c.JSON(http.StatusCreated, types.NewSuccessResponse(space))
}

func (h *SpacesHandler) UpdateSpace(c *gin.Context) {
	spaceID, err := strconv.Atoi(c.Param("spaceId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeValidation, "Invalid space ID"))
		return
	}
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeValidation, err.Error()))
		return
	}

	userID := c.GetInt("userId")
	userRoleID, err := h.spacesRepo.GetUserRoleIDInSpace(userID, spaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}
	if userRoleID == 0 || userRoleID != globals.DefaultOwnerRoleID {
		c.JSON(http.StatusForbidden, types.NewErrorResponse(types.ErrorCodeForbidden, "No permission to edit space"))
		return
	}

	err = h.spacesRepo.UpdateSpaceName(spaceID, req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}
	c.JSON(http.StatusOK, types.NewSuccessResponse(gin.H{"message": "Space updated successfully"}))
}

func (h *SpacesHandler) DeleteSpace(c *gin.Context) {
	spaceID, err := strconv.Atoi(c.Param("spaceId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeValidation, "Invalid space ID"))
		return
	}
	userID := c.GetInt("userId")
	userRoleID, err := h.spacesRepo.GetUserRoleIDInSpace(userID, spaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}
	if userRoleID == 0 || userRoleID != globals.DefaultOwnerRoleID {
		c.JSON(http.StatusForbidden, types.NewErrorResponse(types.ErrorCodeForbidden, "No permission to delete space"))
		return
	}
	err = h.spacesRepo.SetSpaceDeleted(spaceID, true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}
	c.JSON(http.StatusOK, types.NewSuccessResponse(gin.H{"message": "Space deleted successfully"}))
}

func (h *SpacesHandler) RestoreSpace(c *gin.Context) {
	spaceID, err := strconv.Atoi(c.Param("spaceId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeValidation, "Invalid space ID"))
		return
	}
	userID := c.GetInt("userId")
	userRoleID, err := h.spacesRepo.GetUserRoleIDInSpace(userID, spaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}
	if userRoleID == 0 || userRoleID != globals.DefaultOwnerRoleID {
		c.JSON(http.StatusForbidden, types.NewErrorResponse(types.ErrorCodeForbidden, "No permission to restore space"))
		return
	}
	err = h.spacesRepo.SetSpaceDeleted(spaceID, false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}
	c.JSON(http.StatusOK, types.NewSuccessResponse(gin.H{"message": "Space restored successfully"}))
}

func (h *SpacesHandler) InviteUser(c *gin.Context) {
	spaceID, err := strconv.Atoi(c.Param("spaceId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeValidation, "Invalid space ID"))
		return
	}
	userID := c.GetInt("userId")
	userRoleID, err := h.spacesRepo.GetUserRoleIDInSpace(userID, spaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}
	if userRoleID == 0 || userRoleID != globals.DefaultOwnerRoleID {
		c.JSON(http.StatusForbidden, types.NewErrorResponse(types.ErrorCodeForbidden, "No permission to invite user"))
		return
	}
	var req struct {
		Username string `json:"username" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeValidation, err.Error()))
		return
	}

	// Get user by username
	user, err := h.spacesRepo.GetUserByUsername(req.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, "Failed to find user"))
		return
	}
	if user == nil {
		c.JSON(http.StatusNotFound, types.NewErrorResponse(types.ErrorCodeNotFound, "User not found"))
		return
	}

	roleGuest, err := h.rolesRepo.GetRoleByName("guest")
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, "Role not found"))
		return
	}

	err = h.spacesRepo.InviteUserToSpace(user.ID, spaceID, roleGuest.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}
	c.JSON(http.StatusOK, types.NewSuccessResponse(gin.H{"message": "User invited successfully"}))
}

func (h *SpacesHandler) GetAccessibleSpaces(c *gin.Context) {
	userID := c.GetInt("userId")

	// Use standardized pagination
	pagination := types.ParsePaginationParams(c)

	spaces, total, err := h.spacesRepo.GetSpacesForUserPaginated(userID, pagination.Offset, pagination.PageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}

	// Use standardized response with pagination
	response := pagination.BuildResponse(spaces, total)
	c.JSON(http.StatusOK, types.NewSuccessResponse(response))
}

func (h *SpacesHandler) RemoveUser(c *gin.Context) {
	spaceID, err := strconv.Atoi(c.Param("spaceId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeValidation, "Invalid space ID"))
		return
	}
	userID := c.GetInt("userId")
	userRoleID, err := h.spacesRepo.GetUserRoleIDInSpace(userID, spaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}
	if userRoleID == 0 || userRoleID != globals.DefaultOwnerRoleID {
		c.JSON(http.StatusForbidden, types.NewErrorResponse(types.ErrorCodeForbidden, "No permission to remove participants"))
		return
	}

	userToRemoveID, err := strconv.Atoi(c.Param("userId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeValidation, "Invalid user ID"))
		return
	}
	targetRoleID, err := h.spacesRepo.GetUserRoleIDInSpace(userToRemoveID, spaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}
	if targetRoleID == 0 {
		c.JSON(http.StatusNotFound, types.NewErrorResponse(types.ErrorCodeNotFound, "User not found in space"))
		return
	}

	// Prevent removing the owner
	if targetRoleID == globals.DefaultOwnerRoleID {
		c.JSON(http.StatusForbidden, types.NewErrorResponse(types.ErrorCodeForbidden, "Cannot remove owner from space"))
		return
	}

	err = h.spacesRepo.RemoveUserFromSpace(userToRemoveID, spaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}
	c.JSON(http.StatusOK, types.NewSuccessResponse(gin.H{"message": "User removed from space successfully"}))
}

func (h *SpacesHandler) GetUsersInSpace(c *gin.Context) {
	spaceID, err := strconv.Atoi(c.Param("spaceId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeValidation, "Invalid space ID"))
		return
	}
	userID := c.GetInt("userId")
	userRoleID, err := h.spacesRepo.GetUserRoleIDInSpace(userID, spaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}
	if userRoleID == 0 {
		c.JSON(http.StatusForbidden, types.NewErrorResponse(types.ErrorCodeForbidden, "No access to the space"))
		return
	}

	// Use standardized pagination
	pagination := types.ParsePaginationParams(c)

	participants, total, err := h.spacesRepo.GetUsersInSpacePaginated(spaceID, pagination.Offset, pagination.PageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}

	// Use standardized response with pagination
	response := pagination.BuildResponse(participants, total)
	c.JSON(http.StatusOK, types.NewSuccessResponse(response))
}
