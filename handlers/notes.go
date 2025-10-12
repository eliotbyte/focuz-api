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

	"log/slog"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/lib/pq"
)

type NotesHandler struct {
	repo       *repository.NotesRepository
	spacesRepo *repository.SpacesRepository
}

func NewNotesHandler(repo *repository.NotesRepository, spacesRepo *repository.SpacesRepository) *NotesHandler {
	return &NotesHandler{repo: repo, spacesRepo: spacesRepo}
}

func AuthMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, types.NewErrorResponse(types.ErrorCodeUnauthorized, "Authorization header required"))
			c.Abort()
			return
		}
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, types.NewErrorResponse(types.ErrorCodeUnauthorized, "Invalid authorization header"))
			c.Abort()
			return
		}
		token, err := jwt.ParseWithClaims(parts[1], jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, types.NewErrorResponse(types.ErrorCodeInvalidToken, "Invalid token"))
			c.Abort()
			return
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok || !token.Valid {
			c.JSON(http.StatusUnauthorized, types.NewErrorResponse(types.ErrorCodeInvalidToken, "Invalid token claims"))
			c.Abort()
			return
		}
		// Validate issuer and audience for additional hardening
		if claims["iss"] != "focuz-api" || claims["aud"] != "focuz-fe" {
			c.JSON(http.StatusUnauthorized, types.NewErrorResponse(types.ErrorCodeInvalidToken, "Invalid token claims"))
			c.Abort()
			return
		}
		userID, ok := claims["userId"].(float64)
		if !ok {
			c.JSON(http.StatusUnauthorized, types.NewErrorResponse(types.ErrorCodeInvalidToken, "userId not found in token"))
			c.Abort()
			return
		}

		// Structured logging with PII guard: do not log userId in production
		if strings.ToLower(os.Getenv("APP_ENV")) != "production" {
			slog.Info("auth request", "path", c.Request.URL.Path, "userId", int(userID))
		} else {
			slog.Info("auth request", "path", c.Request.URL.Path)
		}

		c.Set("userId", int(userID))
		c.Next()
	}
}

func (h *NotesHandler) Register(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeValidation, err.Error()))
		return
	}
	// Convert username to lowercase for case-insensitive handling
	req.Username = strings.ToLower(req.Username)
	if len(req.Username) < 3 || len(req.Username) > 50 {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeValidation, "Username must be between 3 and 50 characters"))
		return
	}
	// Enforce basic password policy: minimum length 8 characters
	if len(req.Password) < 8 {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeValidation, "Password must be at least 8 characters"))
		return
	}
	user, err := h.repo.CreateUser(req.Username, req.Password)
	if err != nil {
		// Map unique violation to 409 Conflict for duplicate usernames
		if pgErr, ok := err.(*pq.Error); ok && string(pgErr.Code) == "23505" {
			c.JSON(http.StatusConflict, types.NewErrorResponse(types.ErrorCodeConflict, "Username already exists"))
			return
		}
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, "Failed to register user: "+err.Error()))
		return
	}
	c.JSON(http.StatusCreated, types.NewSuccessResponse(user))
}

func (h *NotesHandler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeValidation, err.Error()))
		return
	}
	// Convert username to lowercase for case-insensitive handling
	req.Username = strings.ToLower(req.Username)
	user, err := h.repo.GetUserByUsername(req.Username)
	if err != nil || user == nil {
		c.JSON(http.StatusUnauthorized, types.NewErrorResponse(types.ErrorCodeUnauthorized, "Invalid username or password"))
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, types.NewErrorResponse(types.ErrorCodeUnauthorized, "Invalid username or password"))
		return
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userId": user.ID,
		"exp":    time.Now().Add(time.Hour * 24).Unix(),
		"iss":    "focuz-api",
		"aud":    "focuz-fe",
	})
	tokenString, err := token.SignedString([]byte(c.MustGet("jwtSecret").(string)))
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, "Failed to generate token"))
		return
	}
	c.JSON(http.StatusOK, types.NewSuccessResponse(gin.H{"token": tokenString}))
}

func (h *NotesHandler) CreateNote(c *gin.Context) {
	var req struct {
		Text     string    `json:"text" binding:"required"`
		Date     time.Time `json:"date" binding:"required"`
		SpaceID  int       `json:"spaceId" binding:"required"`
		Tags     []string  `json:"tags"`
		ParentID *int      `json:"parentId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeValidation, err.Error()))
		return
	}

	userID := c.GetInt("userId")
	roleID, err := h.spacesRepo.GetUserRoleIDInSpace(userID, req.SpaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}
	if roleID == 0 {
		c.JSON(http.StatusForbidden, types.NewErrorResponse(types.ErrorCodeForbidden, "No access to the space"))
		return
	}

	dateStr := req.Date.Format(time.RFC3339)
	note, err := h.repo.CreateNote(userID, req.Text, req.Tags, req.ParentID, &dateStr, req.SpaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}

	c.JSON(http.StatusCreated, types.NewSuccessResponse(note))
}

func (h *NotesHandler) DeleteNote(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeValidation, "Invalid ID"))
		return
	}
	note, err := h.repo.GetNoteByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}
	if note == nil {
		c.JSON(http.StatusNotFound, types.NewErrorResponse(types.ErrorCodeNotFound, "Note not found"))
		return
	}
	userID := c.GetInt("userId")
	roleID, err := h.spacesRepo.GetUserRoleIDInSpace(userID, note.SpaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}
	if roleID == 0 {
		c.JSON(http.StatusForbidden, types.NewErrorResponse(types.ErrorCodeForbidden, "No access to the space"))
		return
	}
	if roleID != globals.DefaultOwnerRoleID {
		c.JSON(http.StatusForbidden, types.NewErrorResponse(types.ErrorCodeForbidden, "Guests cannot delete notes"))
		return
	}
	if err := h.repo.UpdateNoteDeleted(id, true); err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}
	c.JSON(http.StatusOK, types.NewSuccessResponse(gin.H{"message": "Note deleted successfully"}))
}

func (h *NotesHandler) RestoreNote(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeValidation, "Invalid ID"))
		return
	}
	note, err := h.repo.GetNoteByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}
	if note == nil {
		c.JSON(http.StatusNotFound, types.NewErrorResponse(types.ErrorCodeNotFound, "Note not found"))
		return
	}
	userID := c.GetInt("userId")
	roleID, err := h.spacesRepo.GetUserRoleIDInSpace(userID, note.SpaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}
	if roleID == 0 {
		c.JSON(http.StatusForbidden, types.NewErrorResponse(types.ErrorCodeForbidden, "No access to the space"))
		return
	}
	if roleID != globals.DefaultOwnerRoleID {
		c.JSON(http.StatusForbidden, types.NewErrorResponse(types.ErrorCodeForbidden, "Guests cannot restore notes"))
		return
	}
	if err := h.repo.UpdateNoteDeleted(id, false); err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}
	c.JSON(http.StatusOK, types.NewSuccessResponse(gin.H{"message": "Note restored successfully"}))
}

func (h *NotesHandler) GetNote(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeValidation, "Invalid ID"))
		return
	}
	note, err := h.repo.GetNoteByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}
	if note == nil {
		c.JSON(http.StatusNotFound, types.NewErrorResponse(types.ErrorCodeNotFound, "Note not found"))
		return
	}
	userID := c.GetInt("userId")
	roleID, err := h.spacesRepo.GetUserRoleIDInSpace(userID, note.SpaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}
	if roleID == 0 {
		c.JSON(http.StatusForbidden, types.NewErrorResponse(types.ErrorCodeForbidden, "No access to the space"))
		return
	}
	if roleID != globals.DefaultOwnerRoleID && note.UserID != userID {
		c.JSON(http.StatusForbidden, types.NewErrorResponse(types.ErrorCodeForbidden, "No access to the note"))
		return
	}

	c.JSON(http.StatusOK, types.NewSuccessResponse(note))
}

func (h *NotesHandler) GetNotes(c *gin.Context) {
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

	// Use standardized pagination
	pagination := types.ParsePaginationParams(c)

	tags := c.QueryArray("tags")
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

	// Parse date filters
	var dateFrom *time.Time
	if dateFromStr := c.Query("dateFrom"); dateFromStr != "" {
		if parsed, err := time.Parse("2006-01-02", dateFromStr); err == nil {
			dateFrom = &parsed
		}
	}
	var dateTo *time.Time
	if dateToStr := c.Query("dateTo"); dateToStr != "" {
		if parsed, err := time.Parse("2006-01-02", dateToStr); err == nil {
			dateTo = &parsed
		}
	}

	sortParam := c.Query("sort")
	sortField := "created_at"
	sortOrder := "DESC"
	if sortParam != "" {
		parts := strings.Split(sortParam, ",")
		if len(parts) == 2 {
			field := strings.ToLower(strings.TrimSpace(parts[0]))
			order := strings.ToUpper(strings.TrimSpace(parts[1]))

			// Normalize supported fields to DB column names
			switch field {
			case "createdat", "created_at":
				sortField = "created_at"
			case "modifiedat", "modified_at":
				sortField = "modified_at"
			}

			if order == "ASC" || order == "DESC" {
				sortOrder = order
			}
		}
	}

	filters := models.NoteFilters{
		Tags:        tags,
		NotReply:    notReply,
		Page:        pagination.Page,
		PageSize:    pagination.PageSize,
		SearchQuery: searchQueryPtr,
		ParentID:    parentID,
		SortField:   sortField,
		SortOrder:   sortOrder,
		DateFrom:    dateFrom,
		DateTo:      dateTo,
	}
	notes, total, err := h.repo.GetNotes(userID, spaceID, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}

	// Use standardized response with pagination
	response := pagination.BuildResponse(notes, total)
	c.JSON(http.StatusOK, types.NewSuccessResponse(response))
}

func (h *NotesHandler) GetTagAutocomplete(c *gin.Context) {
	text := c.Query("text")
	spaceID, err := strconv.Atoi(c.Query("spaceId"))
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

	tags, err := h.repo.GetTagAutocomplete(text, spaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}

	c.JSON(http.StatusOK, types.NewSuccessResponse(tags))
}
