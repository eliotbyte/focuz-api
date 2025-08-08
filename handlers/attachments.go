package handlers

import (
	"context"
	"fmt"
	"focuz-api/initializers"
	"focuz-api/repository"
	"focuz-api/types"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"mime/multipart"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type AttachmentsHandler struct {
	attachmentsRepo *repository.AttachmentsRepository
	notesRepo       *repository.NotesRepository
	spacesRepo      *repository.SpacesRepository
	topicsRepo      *repository.TopicsRepository
}

func NewAttachmentsHandler(a *repository.AttachmentsRepository, n *repository.NotesRepository, s *repository.SpacesRepository, t *repository.TopicsRepository) *AttachmentsHandler {
	return &AttachmentsHandler{attachmentsRepo: a, notesRepo: n, spacesRepo: s, topicsRepo: t}
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

	topic, err := h.topicsRepo.GetTopicByID(note.TopicID)
	if err != nil || topic == nil || topic.IsDeleted {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeInvalidRequest, "invalid topic"))
		return
	}

	roleID, err := h.spacesRepo.GetUserRoleIDInSpace(userID, topic.SpaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}
	if roleID == 0 {
		c.JSON(http.StatusForbidden, types.NewErrorResponse(types.ErrorCodeForbidden, "no access"))
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeValidation, "file is required"))
		return
	}

	// Check if file type is allowed
	if err := initializers.CheckFileAllowed(file.Size, file.Header.Get("Content-Type")); err != nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(types.ErrorCodeValidation, err.Error()))
		return
	}

	// Upload file to MinIO
	attachmentID, err := h.uploadFileToMinIO(file, noteID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}

	c.JSON(http.StatusCreated, types.NewSuccessResponse(map[string]interface{}{
		"attachment_id": attachmentID,
		"filename":      file.Filename,
		"size":          file.Size,
	}))
}

func (h *AttachmentsHandler) uploadFileToMinIO(file *multipart.FileHeader, noteID int) (string, error) {
	// Create attachment record
	attachmentID, err := h.attachmentsRepo.CreateAttachment(noteID, file.Filename, file.Header.Get("Content-Type"), file.Size)
	if err != nil {
		return "", err
	}

	// Open the file
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
			ContentType: file.Header.Get("Content-Type"),
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

	topic, err := h.topicsRepo.GetTopicByID(note.TopicID)
	if err != nil || topic == nil || topic.IsDeleted {
		c.JSON(http.StatusForbidden, types.NewErrorResponse(types.ErrorCodeForbidden, "no access"))
		return
	}

	roleID, err := h.spacesRepo.GetUserRoleIDInSpace(userID, topic.SpaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, err.Error()))
		return
	}
	if roleID == 0 {
		c.JSON(http.StatusForbidden, types.NewErrorResponse(types.ErrorCodeForbidden, "no access"))
		return
	}

	reqParams := url.Values{}
	reqParams.Set("response-content-disposition", fmt.Sprintf("inline; filename=\"%s\"", sanitizeFilename(att.FileName)))

	expiry := initializers.Conf.Expiry

	// Извлекаем только хост без схемы из ExternalEndpoint
	extEndpoint := initializers.Conf.ExternalEndpoint
	if strings.HasPrefix(extEndpoint, "http://") {
		extEndpoint = strings.TrimPrefix(extEndpoint, "http://")
	} else if strings.HasPrefix(extEndpoint, "https://") {
		extEndpoint = strings.TrimPrefix(extEndpoint, "https://")
	}

	externalClient, err := minio.New(extEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(initializers.Conf.AccessKey, initializers.Conf.SecretKey, ""),
		Secure: initializers.Conf.UseSSL,
		Region: "us-east-1", // добавляем регион для корректного расчёта подписи
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, "failed to create external minio client"))
		return
	}

	presignedURL, err := externalClient.PresignedGetObject(
		context.Background(),
		initializers.Conf.Bucket,
		att.ID,
		expiry,
		reqParams,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(types.ErrorCodeInternal, "failed to create presigned url"))
		return
	}

	c.JSON(http.StatusOK, types.NewSuccessResponse(gin.H{
		"url": presignedURL.String(),
	}))
}

func sanitizeFilename(name string) string {
	return strings.ReplaceAll(name, "\"", "")
}
