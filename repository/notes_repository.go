package repository

import (
	"database/sql"
	"focuz-api/models"
	"time"
)

type NotesRepository struct {
	db *sql.DB
}

func NewNotesRepository(db *sql.DB) *NotesRepository {
	return &NotesRepository{db: db}
}

func (r *NotesRepository) CreateNote(text string, tags []string, parentID *int, date *string) (*models.Note, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Вставка заметки
	var noteDate time.Time
	if date != nil {
		noteDate, _ = time.Parse(time.RFC3339, *date)
	} else {
		noteDate = time.Now()
	}
	var noteID int
	err = tx.QueryRow(`
		INSERT INTO note (text, created_at, modified_at, date, parent_id)
		VALUES ($1, NOW(), NOW(), $2, $3)
		RETURNING id`,
		text, noteDate, parentID).Scan(&noteID)
	if err != nil {
		return nil, err
	}

	// Обработка тегов
	for _, tagName := range tags {
		var tagID int
		err := tx.QueryRow(`
			INSERT INTO tag (name) VALUES ($1)
			ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
			RETURNING id`, tagName).Scan(&tagID)
		if err != nil {
			return nil, err
		}
		_, err = tx.Exec(`
			INSERT INTO note_to_tag (note_id, tag_id)
			VALUES ($1, $2)
			ON CONFLICT DO NOTHING`, noteID, tagID)
		if err != nil {
			return nil, err
		}
	}

	// Подсчёт reply_count для родителя
	if parentID != nil {
		_, err = tx.Exec(`
			UPDATE note SET reply_count = reply_count + 1
			WHERE id = $1`, *parentID)
		if err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return r.GetNoteByID(noteID)
}

func (r *NotesRepository) UpdateNoteDeleted(id int, isDeleted bool) error {
	_, err := r.db.Exec(`
		UPDATE note SET is_deleted = $1, modified_at = NOW()
		WHERE id = $2`, isDeleted, id)
	return err
}

func (r *NotesRepository) GetNoteByID(id int) (*models.Note, error) {
	var note models.Note
	var parentID sql.NullInt64
	var parentText sql.NullString
	err := r.db.QueryRow(`
		SELECT n.id, n.text, n.created_at, n.modified_at, n.date, n.parent_id,
		       n.reply_count, n.is_deleted,
		       p.text AS parent_text
		FROM note n
		LEFT JOIN note p ON n.parent_id = p.id
		WHERE n.id = $1`, id).Scan(
		&note.ID, &note.Text, &note.CreatedAt, &note.ModifiedAt, &note.Date,
		&parentID, &note.ReplyCount, &note.IsDeleted, &parentText)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if parentID.Valid {
		note.Parent = &models.ParentNote{
			ID:   int(parentID.Int64),
			Text: truncate(parentText.String, 20),
		}
	}

	rows, err := r.db.Query(`
		SELECT t.name FROM tag t
		JOIN note_to_tag nt ON t.id = nt.tag_id
		WHERE nt.note_id = $1`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, err
		}
		note.Tags = append(note.Tags, tag)
	}

	return &note, nil
}

func (r *NotesRepository) GetNotes(page, pageSize int) ([]*models.Note, int, error) {
	offset := (page - 1) * pageSize
	rows, err := r.db.Query(`
		SELECT id, text, created_at, modified_at, date, parent_id, reply_count
		FROM note
		WHERE is_deleted = FALSE
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var notes []*models.Note
	for rows.Next() {
		var note models.Note
		var parentID sql.NullInt64
		if err := rows.Scan(&note.ID, &note.Text, &note.CreatedAt, &note.ModifiedAt,
			&note.Date, &parentID, &note.ReplyCount); err != nil {
			return nil, 0, err
		}
		if parentID.Valid {
			parent, err := r.GetNoteByID(int(parentID.Int64))
			if err == nil && parent != nil {
				note.Parent = &models.ParentNote{
					ID:   parent.ID,
					Text: truncate(parent.Text, 20),
				}
			}
		}
		notes = append(notes, &note)
	}

	for _, note := range notes {
		rows, err := r.db.Query(`
			SELECT t.name FROM tag t
			JOIN note_to_tag nt ON t.id = nt.tag_id
			WHERE nt.note_id = $1`, note.ID)
		if err != nil {
			return nil, 0, err
		}
		defer rows.Close()
		for rows.Next() {
			var tag string
			if err := rows.Scan(&tag); err != nil {
				return nil, 0, err
			}
			note.Tags = append(note.Tags, tag)
		}
	}

	var total int
	err = r.db.QueryRow(`SELECT COUNT(*) FROM note WHERE is_deleted = FALSE`).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	return notes, total, nil
}

func truncate(s string, max int) string {
	if len(s) > max {
		return s[:max]
	}
	return s
}
