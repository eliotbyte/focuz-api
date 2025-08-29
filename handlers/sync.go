package handlers

import (
	"net/http"
	"strconv"
	"time"

	"focuz-api/repository"
	"focuz-api/types"

	"github.com/gin-gonic/gin"
)

type SyncHandler struct {
	syncRepo    *repository.SyncRepository
	spacesRepo  *repository.SpacesRepository
	tagsRepo    *repository.TagsRepository
	filtersRepo *repository.FiltersRepository
}

func NewSyncHandler(syncRepo *repository.SyncRepository, spacesRepo *repository.SpacesRepository, tagsRepo *repository.TagsRepository, filtersRepo *repository.FiltersRepository) *SyncHandler {
	return &SyncHandler{syncRepo: syncRepo, spacesRepo: spacesRepo, tagsRepo: tagsRepo, filtersRepo: filtersRepo}
}

// GET /sync?since=RFC3339[&spaceId=]
func (h *SyncHandler) Pull(c *gin.Context) {
	sinceStr := c.Query("since")
	if sinceStr == "" {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeValidation, "since is required (RFC3339)"))
		return
	}
	since, err := time.Parse(time.RFC3339, sinceStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeValidation, "since must be RFC3339"))
		return
	}
	userID := c.GetInt("userId")
	spaceIDParam := c.Query("spaceId")
	var spaceIDs []int
	if spaceIDParam != "" {
		id, err := strconv.Atoi(spaceIDParam)
		if err != nil {
			c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeValidation, "invalid spaceId"))
			return
		}
		roleID, err := h.spacesRepo.GetUserRoleIDInSpace(userID, id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
			return
		}
		if roleID == 0 {
			c.JSON(http.StatusForbidden, types.NewErrorResponse(types.ErrorCodeForbidden, "No access to the space"))
			return
		}
		spaceIDs = []int{id}
	} else {
		spaces, err := h.spacesRepo.GetSpacesForUser(userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
			return
		}
		for _, s := range spaces {
			spaceIDs = append(spaceIDs, s.ID)
		}
	}
	changes, err := h.syncRepo.GetChangesSince(userID, spaceIDs, since)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}
	c.JSON(http.StatusOK, types.NewSuccessResponse(changes))
}

// POST /sync
func (h *SyncHandler) Push(c *gin.Context) {
	var req types.SyncPushRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeValidation, err.Error()))
		return
	}
	userID := c.GetInt("userId")
	res, err := h.syncRepo.ApplyChanges(userID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}
	c.JSON(http.StatusOK, types.NewSuccessResponse(res))
}

// GET /spaces/:spaceId/tags
func (h *SyncHandler) GetTagsBySpace(c *gin.Context) {
	spaceID, err := strconv.Atoi(c.Param("spaceId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeValidation, "invalid spaceId"))
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
	tags, err := h.tagsRepo.GetTagsBySpace(spaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}
	c.JSON(http.StatusOK, types.NewSuccessResponse(tags))
}

// GET /spaces/:spaceId/filters
func (h *SyncHandler) GetFiltersBySpace(c *gin.Context) {
	spaceID, err := strconv.Atoi(c.Param("spaceId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeValidation, "invalid spaceId"))
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
	// Reuse filters repo list with default pagination
	items, total, err := h.filtersRepo.List(spaceID, 1, 1000)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}
	_ = total
	c.JSON(http.StatusOK, types.NewSuccessResponse(items))
}
