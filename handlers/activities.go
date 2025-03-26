package handlers

import (
	"encoding/json"
	"errors"
	"focuz-api/models"
	"focuz-api/repository"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type ActivitiesHandler struct {
	activitiesRepo    *repository.ActivitiesRepository
	spacesRepo        *repository.SpacesRepository
	topicsRepo        *repository.TopicsRepository
	notesRepo         *repository.NotesRepository
	activityTypesRepo *repository.ActivityTypesRepository
}

func NewActivitiesHandler(
	ar *repository.ActivitiesRepository,
	sr *repository.SpacesRepository,
	tr *repository.TopicsRepository,
	nr *repository.NotesRepository,
	atr *repository.ActivityTypesRepository,
) *ActivitiesHandler {
	return &ActivitiesHandler{
		activitiesRepo:    ar,
		spacesRepo:        sr,
		topicsRepo:        tr,
		notesRepo:         nr,
		activityTypesRepo: atr,
	}
}

func (h *ActivitiesHandler) CreateActivity(c *gin.Context) {
	var req struct {
		TypeID int    `json:"typeId"`
		Value  string `json:"value"`
		NoteID *int   `json:"note_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	activityType, err := h.activityTypesRepo.GetActivityTypeByID(req.TypeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if activityType == nil || activityType.IsDeleted {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or deleted activity type"})
		return
	}
	userID := c.GetInt("userId")

	var spaceID int
	if req.NoteID != nil {
		note, nerr := h.notesRepo.GetNoteByID(*req.NoteID)
		if nerr != nil || note == nil || note.IsDeleted {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid note"})
			return
		}
		topic, terr := h.topicsRepo.GetTopicByID(note.TopicID)
		if terr != nil || topic == nil || topic.IsDeleted {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid topic"})
			return
		}
		spaceID = topic.SpaceID
		roleID, rerr := h.spacesRepo.GetUserRoleIDInSpace(userID, spaceID)
		if rerr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": rerr.Error()})
			return
		}
		if roleID == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "No access to this note"})
			return
		}
	}

	checkedValue, err := h.validateActivityValue(activityType, req.Value)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	created, err := h.activitiesRepo.CreateActivity(userID, req.TypeID, checkedValue, req.NoteID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, created)
}

func (h *ActivitiesHandler) DeleteActivity(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("activityId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid activity ID"})
		return
	}
	activity, err := h.activitiesRepo.GetActivityByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if activity == nil || activity.IsDeleted {
		c.JSON(http.StatusNotFound, gin.H{"error": "Activity not found"})
		return
	}
	userID := c.GetInt("userId")
	spaceID, perr := h.getSpaceIDForActivity(activity)
	if perr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": perr.Error()})
		return
	}
	if spaceID > 0 {
		roleID, rerr := h.spacesRepo.GetUserRoleIDInSpace(userID, spaceID)
		if rerr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": rerr.Error()})
			return
		}
		if roleID == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "No access to this activity"})
			return
		}
	}
	err = h.activitiesRepo.SetActivityDeleted(id, true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *ActivitiesHandler) RestoreActivity(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("activityId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid activity ID"})
		return
	}
	activity, err := h.activitiesRepo.GetActivityByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if activity == nil || !activity.IsDeleted {
		c.JSON(http.StatusNotFound, gin.H{"error": "Activity not found"})
		return
	}
	userID := c.GetInt("userId")
	spaceID, perr := h.getSpaceIDForActivity(activity)
	if perr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": perr.Error()})
		return
	}
	if spaceID > 0 {
		roleID, rerr := h.spacesRepo.GetUserRoleIDInSpace(userID, spaceID)
		if rerr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": rerr.Error()})
			return
		}
		if roleID == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "No access to this activity"})
			return
		}
	}
	err = h.activitiesRepo.SetActivityDeleted(id, false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *ActivitiesHandler) UpdateActivity(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("activityId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid activity ID"})
		return
	}
	activity, err := h.activitiesRepo.GetActivityByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if activity == nil || activity.IsDeleted {
		c.JSON(http.StatusNotFound, gin.H{"error": "Activity not found"})
		return
	}
	var req struct {
		Value  string `json:"value"`
		NoteID *int   `json:"note_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	activityType, err := h.activityTypesRepo.GetActivityTypeByID(activity.TypeID)
	if err != nil || activityType == nil || activityType.IsDeleted {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid activity type"})
		return
	}
	userID := c.GetInt("userId")

	var spaceID int
	if req.NoteID != nil && *req.NoteID > 0 {
		note, nerr := h.notesRepo.GetNoteByID(*req.NoteID)
		if nerr != nil || note == nil || note.IsDeleted {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid note"})
			return
		}
		topic, terr := h.topicsRepo.GetTopicByID(note.TopicID)
		if terr != nil || topic == nil || topic.IsDeleted {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid topic"})
			return
		}
		spaceID = topic.SpaceID
		roleID, rerr := h.spacesRepo.GetUserRoleIDInSpace(userID, spaceID)
		if rerr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": rerr.Error()})
			return
		}
		if roleID == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "No access to this note"})
			return
		}
	} else {
		spaceID, _ = h.getSpaceIDForActivity(activity)
		if spaceID > 0 {
			roleID, rerr := h.spacesRepo.GetUserRoleIDInSpace(userID, spaceID)
			if rerr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": rerr.Error()})
				return
			}
			if roleID == 0 {
				c.JSON(http.StatusForbidden, gin.H{"error": "No access to this activity"})
				return
			}
		}
	}

	checkedValue, err := h.validateActivityValue(activityType, req.Value)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err = h.activitiesRepo.UpdateActivity(id, checkedValue, req.NoteID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *ActivitiesHandler) validateActivityValue(t *models.ActivityType, raw string) ([]byte, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, errors.New("empty value")
	}
	switch t.ValueType {
	case "integer":
		v, err := strconv.Atoi(raw)
		if err != nil {
			return nil, errors.New("value must be integer")
		}
		if t.MinValue != nil && float64(v) < *t.MinValue {
			return nil, errors.New("value is out of range")
		}
		if t.MaxValue != nil && float64(v) > *t.MaxValue {
			return nil, errors.New("value is out of range")
		}
		m := map[string]any{"data": v}
		return json.Marshal(m)
	case "float":
		f, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return nil, errors.New("value must be float")
		}
		if t.MinValue != nil && f < *t.MinValue {
			return nil, errors.New("value is out of range")
		}
		if t.MaxValue != nil && f > *t.MaxValue {
			return nil, errors.New("value is out of range")
		}
		m := map[string]any{"data": f}
		return json.Marshal(m)
	case "boolean":
		b, err := strconv.ParseBool(raw)
		if err != nil {
			return nil, errors.New("value must be boolean")
		}
		m := map[string]any{"data": b}
		return json.Marshal(m)
	case "text":
		m := map[string]any{"data": raw}
		return json.Marshal(m)
	case "time":
		_, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return nil, errors.New("value must be valid RFC3339 time")
		}
		m := map[string]any{"data": raw}
		return json.Marshal(m)
	default:
		return nil, errors.New("unsupported value type")
	}
}

func (h *ActivitiesHandler) getSpaceIDForActivity(activity *models.Activity) (int, error) {
	if activity.NoteID == nil {
		return 0, nil
	}
	note, err := h.notesRepo.GetNoteByID(*activity.NoteID)
	if err != nil || note == nil || note.IsDeleted {
		return 0, errors.New("invalid note")
	}
	topic, err := h.topicsRepo.GetTopicByID(note.TopicID)
	if err != nil || topic == nil || topic.IsDeleted {
		return 0, errors.New("invalid topic")
	}
	return topic.SpaceID, nil
}
