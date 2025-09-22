package handlers

import (
	"context"
	"focuz-api/initializers"
	"focuz-api/repository"
	"focuz-api/types"
	"net/http"
	"strconv"
	"strings"

	"mime/multipart"

	"github.com/gabriel-vasile/mimetype"
	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
)

type AttachmentsHandler struct {
	attachmentsRepo *repository.AttachmentsRepository
	notesRepo       *repository.NotesRepository
	spacesRepo      *repository.SpacesRepository
}

func NewAttachmentsHandler(a *repository.AttachmentsRepository, n *repository.NotesRepository, s *repository.SpacesRepository) *AttachmentsHandler {
	return &AttachmentsHandler{attachmentsRepo: a, notesRepo: n, spacesRepo: s}
}

func (h *AttachmentsHandler) UploadFile(c *gin.Context) {
	userID := c.GetInt("userId")

	noteIDStr := c.PostForm("note_id")
	if noteIDStr == "" {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeValidation, "note_id is required"))
		return
	}
	noteID, err := strconv.Atoi(noteIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeValidation, "invalid note_id"))
		return
	}

	note, err := h.notesRepo.GetNoteByID(noteID)
	if err != nil || note == nil || note.IsDeleted {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeInvalidRequest, "invalid note"))
		return
	}

	roleID, err := h.spacesRepo.GetUserRoleIDInSpace(userID, note.SpaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}
	if roleID == 0 {
		c.JSON(http.StatusForbidden, types.NewErrorResponse(types.ErrorCodeForbidden, "no access"))
		return
	}

	// Limit request body size before reading multipart data
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, initializers.Conf.MaxSize)

	file, err := c.FormFile("file")
	if err != nil {
		// If the client sent a body larger than allowed limit, return 413
		if strings.Contains(strings.ToLower(err.Error()), "request body too large") {
			c.JSON(http.StatusRequestEntityTooLarge, types.NewErrorResponse(types.ErrorCodeValidation, "file size exceeds the limit"))
			return
		}
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeValidation, "file is required"))
		return
	}

	// Detect real MIME type from file content, not from client header
	sniff, openErr := file.Open()
	if openErr != nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeValidation, "cannot open uploaded file"))
		return
	}
	mt, detectErr := mimetype.DetectReader(sniff)
	_ = sniff.Close()
	if detectErr != nil || mt == nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeValidation, "failed to detect file type"))
		return
	}
	detectedCT := initializers.Conf.FileTypes[0]
	// use base MIME from detection
	detectedCT = strings.Split(mt.String(), ";")[0]

	// Validate against server-side policy
	if err := initializers.CheckFileAllowed(file.Size, detectedCT); err != nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeValidation, err.Error()))
		return
	}

	// Upload file to MinIO using detected content type
	attachmentID, err := h.uploadFileToMinIO(file, noteID, detectedCT)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}

	// Touch note.modified_at to reflect attachment change
	_ = h.notesRepo.TouchNoteModified(noteID)

	c.JSON(http.StatusCreated, types.NewSuccessResponse(map[string]interface{}{
		"attachment_id": attachmentID,
		"filename":      file.Filename,
		"size":          file.Size,
	}))
}

func (h *AttachmentsHandler) uploadFileToMinIO(file *multipart.FileHeader, noteID int, contentType string) (string, error) {
	// Create attachment record with server-detected content type
	attachmentID, err := h.attachmentsRepo.CreateAttachment(noteID, file.Filename, contentType, file.Size)
	if err != nil {
		return "", err
	}

	// Open the file (fresh reader after detection)
	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	// Upload to MinIO
	_, err = initializers.MinioClient.PutObject(
		context.Background(),
		initializers.Conf.Bucket,
		attachmentID,
		src,
		file.Size,
		minio.PutObjectOptions{
			ContentType: contentType,
		},
	)
	if err != nil {
		return "", err
	}

	return attachmentID, nil
}

func (h *AttachmentsHandler) GetFile(c *gin.Context) {
	userID := c.GetInt("userId")
	attID := c.Param("id")
	if attID == "" {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeValidation, "attachment id is required"))
		return
	}

	att, err := h.attachmentsRepo.GetAttachmentByID(attID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}
	if att == nil {
		c.JSON(http.StatusNotFound, types.NewErrorResponse(types.ErrorCodeNotFound, "attachment not found"))
		return
	}

	note, err := h.notesRepo.GetNoteByID(att.NoteID)
	if err != nil || note == nil || note.IsDeleted {
		c.JSON(http.StatusForbidden, types.NewErrorResponse(types.ErrorCodeForbidden, "no access"))
		return
	}

	roleID, err := h.spacesRepo.GetUserRoleIDInSpace(userID, note.SpaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}
	if roleID == 0 {
		c.JSON(http.StatusForbidden, types.NewErrorResponse(types.ErrorCodeForbidden, "no access"))
		return
	}

	url, err := initializers.GenerateAttachmentURL(att.ID, att.FileName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, "failed to create presigned url"))
		return
	}

	c.JSON(http.StatusOK, types.NewSuccessResponse(gin.H{
		"url": url,
	}))
}
