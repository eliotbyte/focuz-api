package handlers

import (
	"focuz-api/globals"
	"focuz-api/repository"
	"focuz-api/types"
	"net/http"
	"strconv"
	"strings"
	"time"

	"focuz-api/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type NotesHandler struct {
	repo       *repository.NotesRepository
	spacesRepo *repository.SpacesRepository
	topicsRepo *repository.TopicsRepository
}

func NewNotesHandler(repo *repository.NotesRepository, spacesRepo *repository.SpacesRepository, topicsRepo *repository.TopicsRepository) *NotesHandler {
	return &NotesHandler{repo: repo, spacesRepo: spacesRepo, topicsRepo: topicsRepo}
}

func AuthMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header"})
			c.Abort()
			return
		}
		token, err := jwt.Parse(parts[1], func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			c.Abort()
			return
		}
		userID, ok := claims["userId"].(float64)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "userId not found in token"})
			c.Abort()
			return
		}
		c.Set("userId", int(userID))
		c.Next()
	}
}

func (h *NotesHandler) Register(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(req.Username) < 3 || len(req.Username) > 50 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username must be between 3 and 50 characters"})
		return
	}
	user, err := h.repo.CreateUser(req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register user: " + err.Error()})
		return
	}
	c.JSON(http.StatusCreated, user)
}

func (h *NotesHandler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	user, err := h.repo.GetUserByUsername(req.Username)
	if err != nil || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userId": user.ID,
		"exp":    time.Now().Add(time.Hour * 24).Unix(),
	})
	tokenString, err := token.SignedString([]byte(c.MustGet("jwtSecret").(string)))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": tokenString})
}

func (h *NotesHandler) CreateNote(c *gin.Context) {
	var req struct {
		Text    string    `json:"text"`
		Date    time.Time `json:"date"`
		TopicID int       `json:"topicId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	topic, err := h.topicsRepo.GetTopicByID(req.TopicID)
	if err != nil || topic == nil || topic.IsDeleted {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid topic"})
		return
	}

	topicType := types.GetTopicTypeByID(topic.TypeID)
	if topicType == nil || topicType.Name != "notebook" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Notes can only be created in notebook topics"})
		return
	}

	userID := c.GetInt("userId")
	roleID, err := h.spacesRepo.GetUserRoleIDInSpace(userID, topic.SpaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if roleID == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "No access to the space"})
		return
	}

	dateStr := req.Date.Format(time.RFC3339)
	note, err := h.repo.CreateNote(userID, req.Text, nil, nil, &dateStr, req.TopicID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, note)
}

func (h *NotesHandler) DeleteNote(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}
	note, err := h.repo.GetNoteByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if note == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Note not found"})
		return
	}
	userID := c.GetInt("userId")
	topic, err := h.topicsRepo.GetTopicByID(note.TopicID)
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
	if roleID != globals.DefaultOwnerRoleID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Guests cannot delete notes"})
		return
	}
	if err := h.repo.UpdateNoteDeleted(id, true); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *NotesHandler) RestoreNote(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}
	note, err := h.repo.GetNoteByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if note == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Note not found"})
		return
	}
	userID := c.GetInt("userId")
	topic, err := h.topicsRepo.GetTopicByID(note.TopicID)
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
	if roleID != globals.DefaultOwnerRoleID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Guests cannot restore notes"})
		return
	}
	if err := h.repo.UpdateNoteDeleted(id, false); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *NotesHandler) GetNote(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}
	note, err := h.repo.GetNoteByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if note == nil || note.IsDeleted {
		c.JSON(http.StatusNotFound, gin.H{"error": "Note not found"})
		return
	}
	userID := c.GetInt("userId")
	topic, err := h.topicsRepo.GetTopicByID(note.TopicID)
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
	c.JSON(http.StatusOK, note)
}

func (h *NotesHandler) GetNotes(c *gin.Context) {
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
	includeTags := c.QueryArray("includeTags")
	excludeTags := c.QueryArray("excludeTags")
	notReplyParam := c.Query("notReply")
	notReply := false
	if strings.ToLower(notReplyParam) == "true" {
		notReply = true
	}
	searchQuery := c.Query("search")
	var searchQueryPtr *string
	if searchQuery != "" {
		searchQueryPtr = &searchQuery
	}
	parentIDParam := c.Query("parentId")
	var parentID *int
	if parentIDParam != "" {
		tmp, err := strconv.Atoi(parentIDParam)
		if err == nil {
			parentID = &tmp
		}
	}
	filters := models.NoteFilters{
		IncludeTags: includeTags,
		ExcludeTags: excludeTags,
		NotReply:    notReply,
		Page:        page,
		PageSize:    pageSize,
		SearchQuery: searchQueryPtr,
		ParentID:    parentID,
	}
	notes, total, err := h.repo.GetNotes(userID, spaceID, topicID, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"notes": notes,
		"total": total,
	})
}
