package handlers

import (
	"context"
	"fmt"
	"focuz-api/initializers"
	"focuz-api/repository"
	"net/http"
	"net/url"
	"strconv"
	"strings"

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
		c.JSON(http.StatusBadRequest, gin.H{"error": "note_id is required"})
		return
	}
	noteID, err := strconv.Atoi(noteIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid note_id"})
		return
	}

	note, err := h.notesRepo.GetNoteByID(noteID)
	if err != nil || note == nil || note.IsDeleted {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid note"})
		return
	}

	topic, err := h.topicsRepo.GetTopicByID(note.TopicID)
	if err != nil || topic == nil || topic.IsDeleted {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid topic"})
		return
	}

	roleID, err := h.spacesRepo.GetUserRoleIDInSpace(userID, topic.SpaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if roleID == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "no access"})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	fileSize := header.Size

	if err := initializers.CheckFileAllowed(fileSize, contentType); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	attID, err := h.attachmentsRepo.CreateAttachment(noteID, header.Filename, contentType, fileSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	_, err = initializers.MinioClient.PutObject(
		context.Background(),
		initializers.Conf.Bucket,
		attID,
		file,
		fileSize,
		minio.PutObjectOptions{
			ContentType: contentType,
		},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upload to minio"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"attachment_id": attID})
}

func (h *AttachmentsHandler) GetFile(c *gin.Context) {
	userID := c.GetInt("userId")
	attID := c.Param("id")
	if attID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "attachment id is required"})
		return
	}

	att, err := h.attachmentsRepo.GetAttachmentByID(attID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if att == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "attachment not found"})
		return
	}

	note, err := h.notesRepo.GetNoteByID(att.NoteID)
	if err != nil || note == nil || note.IsDeleted {
		c.JSON(http.StatusForbidden, gin.H{"error": "no access"})
		return
	}

	topic, err := h.topicsRepo.GetTopicByID(note.TopicID)
	if err != nil || topic == nil || topic.IsDeleted {
		c.JSON(http.StatusForbidden, gin.H{"error": "no access"})
		return
	}

	roleID, err := h.spacesRepo.GetUserRoleIDInSpace(userID, topic.SpaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if roleID == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "no access"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create external minio client"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create presigned url"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"url": presignedURL.String(),
	})
}

func sanitizeFilename(name string) string {
	return strings.ReplaceAll(name, "\"", "")
}
