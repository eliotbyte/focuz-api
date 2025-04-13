package repository

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type Attachment struct {
	ID         string
	NoteID     int
	FileName   string
	FileType   string
	FileSize   int64
	CreatedAt  time.Time
	ModifiedAt time.Time
}

type AttachmentsRepository struct {
	db *sql.DB
}

func NewAttachmentsRepository(db *sql.DB) *AttachmentsRepository {
	return &AttachmentsRepository{db: db}
}

func (r *AttachmentsRepository) CreateAttachment(noteID int, fileName, fileType string, fileSize int64) (string, error) {
	id := uuid.NewString()
	_, err := r.db.Exec(`
		INSERT INTO attachments (id, note_id, file_name, file_type, file_size, created_at, modified_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
	`, id, noteID, fileName, fileType, fileSize)
	if err != nil {
		return "", err
	}
	return id, nil
}

func (r *AttachmentsRepository) GetAttachmentByID(attID string) (*Attachment, error) {
	var a Attachment
	err := r.db.QueryRow(`
		SELECT id, note_id, file_name, file_type, file_size, created_at, modified_at
		FROM attachments
		WHERE id = $1
	`, attID).Scan(
		&a.ID, &a.NoteID, &a.FileName, &a.FileType, &a.FileSize, &a.CreatedAt, &a.ModifiedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}
