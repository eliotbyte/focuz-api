package repository

import (
	"database/sql"
	"focuz-api/initializers"
	"focuz-api/models"
	"strconv"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type NotesRepository struct {
	db *sql.DB
}

func NewNotesRepository(db *sql.DB) *NotesRepository {
	return &NotesRepository{db: db}
}

func (r *NotesRepository) CreateUser(username, password string) (*models.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	var user models.User
	err = r.db.QueryRow(`
		INSERT INTO users (username, password_hash)
		VALUES ($1, $2)
		RETURNING id, username, created_at
	`, username, string(hash)).Scan(&user.ID, &user.Username, &user.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *NotesRepository) GetUserByUsername(username string) (*models.User, error) {
	var user models.User
	err := r.db.QueryRow(`
		SELECT id, username, password_hash, created_at
		FROM users
		WHERE username = $1
	`, username).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *NotesRepository) CreateNote(userID int, text string, tags []string, parentID *int, date *string, topicID int) (*models.Note, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Get space_id for the topic
	var spaceID int
	err = tx.QueryRow("SELECT space_id FROM topic WHERE id = $1", topicID).Scan(&spaceID)
	if err != nil {
		return nil, err
	}

	var noteDate time.Time
	if date != nil {
		parsed, parseErr := time.Parse(time.RFC3339, *date)
		if parseErr == nil {
			noteDate = parsed
		} else {
			noteDate = time.Now()
		}
	} else {
		noteDate = time.Now()
	}

	var noteID int
	err = tx.QueryRow(`
		INSERT INTO note (user_id, text, created_at, modified_at, date, parent_id, topic_id)
		VALUES ($1, $2, NOW(), NOW(), $3, $4, $5)
		RETURNING id
	`, userID, text, noteDate, parentID, topicID).Scan(&noteID)
	if err != nil {
		return nil, err
	}

	// Process tags if provided
	if tags != nil {
		for _, tagName := range tags {
			var tagID int
			err = tx.QueryRow(`
				INSERT INTO tag (name) VALUES ($1)
				ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
				RETURNING id
			`, tagName).Scan(&tagID)
			if err != nil {
				return nil, err
			}

			// Create note_to_tag entry
			_, err = tx.Exec(`
				INSERT INTO note_to_tag (note_id, tag_id)
				VALUES ($1, $2)
				ON CONFLICT DO NOTHING
			`, noteID, tagID)
			if err != nil {
				return nil, err
			}

			// Create tag_to_space_topic entry
			_, err = tx.Exec(`
				INSERT INTO tag_to_space_topic (tag_id, space_id, topic_id)
				VALUES ($1, $2, $3)
				ON CONFLICT (tag_id, space_id, topic_id) DO NOTHING
			`, tagID, spaceID, topicID)
			if err != nil {
				return nil, err
			}
		}
	}

	if parentID != nil {
		// Get all parent IDs in the chain
		var parentIDs []int
		currentID := *parentID
		for {
			var parentID sql.NullInt64
			err = tx.QueryRow(`
				SELECT parent_id FROM note WHERE id = $1
			`, currentID).Scan(&parentID)
			if err != nil {
				return nil, err
			}

			parentIDs = append(parentIDs, currentID)

			if !parentID.Valid {
				break
			}
			currentID = int(parentID.Int64)
		}

		// Update reply_count for all parents
		for _, pid := range parentIDs {
			_, err = tx.Exec(`
				UPDATE note SET reply_count = reply_count + 1
				WHERE id = $1
			`, pid)
			if err != nil {
				return nil, err
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	// Get the complete note with tags
	note, err := r.GetNoteByID(noteID)
	if err != nil {
		return nil, err
	}

	return note, nil
}

func (r *NotesRepository) UpdateNoteDeleted(id int, isDeleted bool) error {
	_, err := r.db.Exec(`
		UPDATE note SET is_deleted = $1, modified_at = NOW()
		WHERE id = $2
	`, isDeleted, id)
	return err
}

func (r *NotesRepository) GetNoteByID(id int) (*models.Note, error) {
	var note models.Note
	var parentID sql.NullInt64
	var parentText sql.NullString
	err := r.db.QueryRow(`
		SELECT n.id, n.user_id, n.text, n.created_at, n.modified_at, n.date,
		       n.parent_id, n.reply_count, n.is_deleted, n.topic_id,
		       p.text AS parent_text
		FROM note n
		LEFT JOIN note p ON n.parent_id = p.id
		WHERE n.id = $1
	`, id).Scan(
		&note.ID,
		&note.UserID,
		&note.Text,
		&note.CreatedAt,
		&note.ModifiedAt,
		&note.Date,
		&parentID,
		&note.ReplyCount,
		&note.IsDeleted,
		&note.TopicID,
		&parentText,
	)
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
	tagRows, err := r.db.Query(`
		SELECT t.name FROM tag t
		JOIN note_to_tag nt ON t.id = nt.tag_id
		WHERE nt.note_id = $1
	`, id)
	if err != nil {
		return nil, err
	}
	defer tagRows.Close()
	for tagRows.Next() {
		var tag string
		if err := tagRows.Scan(&tag); err != nil {
			return nil, err
		}
		note.Tags = append(note.Tags, tag)
	}
	activities, err := r.getActivitiesForNote(note.ID)
	if err != nil {
		return nil, err
	}
	note.Activities = activities
	attRows, err := r.db.Query(`
		SELECT id, file_name, file_type, file_size
		FROM attachments
		WHERE note_id = $1
	`, note.ID)
	if err != nil {
		return nil, err
	}
	defer attRows.Close()
	var attachments []models.Attachment
	for attRows.Next() {
		var att models.Attachment
		if err := attRows.Scan(&att.ID, &att.FileName, &att.FileType, &att.FileSize); err != nil {
			return nil, err
		}
		url, err := initializers.GenerateAttachmentURL(att.ID, att.FileName)
		if err != nil {
			return nil, err
		}
		att.URL = url
		attachments = append(attachments, att)
	}
	note.Attachments = attachments
	return &note, nil
}

func (r *NotesRepository) GetNotes(userID, spaceID int, topicID *int, filters models.NoteFilters) ([]*models.Note, int, error) {
	offset := (filters.Page - 1) * filters.PageSize
	var conditions []string
	var params []interface{}
	idx := 1
	conditions = append(conditions, "n.is_deleted = FALSE")
	conditions = append(conditions, "t.space_id = $"+strconv.Itoa(idx))
	params = append(params, spaceID)
	idx++
	if topicID != nil {
		conditions = append(conditions, "n.topic_id = $"+strconv.Itoa(idx))
		params = append(params, *topicID)
		idx++
	}
	if filters.ParentID != nil {
		conditions = append(conditions, "n.parent_id = $"+strconv.Itoa(idx))
		params = append(params, *filters.ParentID)
		idx++
	} else if filters.NotReply {
		conditions = append(conditions, "n.parent_id IS NULL")
	}
	if filters.SearchQuery != nil && *filters.SearchQuery != "" {
		conditions = append(conditions, "n.text &@ $"+strconv.Itoa(idx))
		params = append(params, *filters.SearchQuery)
		idx++
	}
	query := `
		SELECT n.id, n.user_id, n.text, n.created_at, n.modified_at, n.date, 
		       n.parent_id, n.reply_count, n.is_deleted, n.topic_id
		FROM note n
		INNER JOIN topic t ON n.topic_id = t.id
	`
	if len(filters.IncludeTags) > 0 {
		for _, tagVal := range filters.IncludeTags {
			query += " INNER JOIN note_to_tag nt_" + tagVal + " ON nt_" + tagVal + ".note_id = n.id " +
				" INNER JOIN tag tg_" + tagVal + " ON tg_" + tagVal + ".id = nt_" + tagVal + ".tag_id AND tg_" + tagVal + ".name = '" + tagVal + "' "
		}
	}
	if len(conditions) > 0 {
		query += " WHERE " + joinConditions(conditions, " AND ")
	}
	if len(filters.ExcludeTags) > 0 {
		for _, exTag := range filters.ExcludeTags {
			query += " AND NOT EXISTS (SELECT 1 FROM note_to_tag xnt INNER JOIN tag xt ON xt.id = xnt.tag_id WHERE xnt.note_id = n.id AND xt.name = '" + exTag + "')"
		}
	}

	sortField := "n.created_at"
	if filters.SortField == "modifiedat" {
		sortField = "n.modified_at"
	}
	query += " ORDER BY " + sortField + " " + filters.SortOrder
	query += " LIMIT $" + strconv.Itoa(idx) + " OFFSET $" + strconv.Itoa(idx+1)
	params = append(params, filters.PageSize, offset)
	rows, err := r.db.Query(query, params...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var notes []*models.Note
	for rows.Next() {
		var note models.Note
		var parentID sql.NullInt64
		err := rows.Scan(
			&note.ID,
			&note.UserID,
			&note.Text,
			&note.CreatedAt,
			&note.ModifiedAt,
			&note.Date,
			&parentID,
			&note.ReplyCount,
			&note.IsDeleted,
			&note.TopicID,
		)
		if err != nil {
			return nil, 0, err
		}
		if parentID.Valid {
			parent, _ := r.GetNoteByID(int(parentID.Int64))
			if parent != nil {
				note.Parent = &models.ParentNote{
					ID:   parent.ID,
					Text: truncate(parent.Text, 20),
				}
			}
		}
		tRows, err := r.db.Query(`
			SELECT t.name FROM tag t
			JOIN note_to_tag nt ON t.id = nt.tag_id
			WHERE nt.note_id = $1
		`, note.ID)
		if err != nil {
			return nil, 0, err
		}
		for tRows.Next() {
			var tg string
			if err := tRows.Scan(&tg); err != nil {
				tRows.Close()
				return nil, 0, err
			}
			note.Tags = append(note.Tags, tg)
		}
		tRows.Close()
		activities, err := r.getActivitiesForNote(note.ID)
		if err != nil {
			return nil, 0, err
		}
		note.Activities = activities
		aRows, err := r.db.Query(`
			SELECT id, file_name, file_type, file_size
			FROM attachments
			WHERE note_id = $1
		`, note.ID)
		if err != nil {
			return nil, 0, err
		}
		var attachments []models.Attachment
		for aRows.Next() {
			var att models.Attachment
			if err := aRows.Scan(&att.ID, &att.FileName, &att.FileType, &att.FileSize); err != nil {
				aRows.Close()
				return nil, 0, err
			}
			url, err := initializers.GenerateAttachmentURL(att.ID, att.FileName)
			if err != nil {
				aRows.Close()
				return nil, 0, err
			}
			att.URL = url
			attachments = append(attachments, att)
		}
		aRows.Close()
		note.Attachments = attachments
		notes = append(notes, &note)
	}
	var total int
	countQuery := `
		SELECT COUNT(n.id)
		FROM note n
		INNER JOIN topic t ON n.topic_id = t.id
	`
	if len(filters.IncludeTags) > 0 {
		for _, tagVal := range filters.IncludeTags {
			countQuery += " INNER JOIN note_to_tag nt_" + tagVal + " ON nt_" + tagVal + ".note_id = n.id " +
				" INNER JOIN tag tg_" + tagVal + " ON tg_" + tagVal + ".id = nt_" + tagVal + ".tag_id AND tg_" + tagVal + ".name = '" + tagVal + "' "
		}
	}
	if len(conditions) > 0 {
		countQuery += " WHERE " + joinConditions(conditions, " AND ")
	}
	if len(filters.ExcludeTags) > 0 {
		for _, exTag := range filters.ExcludeTags {
			countQuery += " AND NOT EXISTS (SELECT 1 FROM note_to_tag xnt INNER JOIN tag xt ON xt.id = xnt.tag_id WHERE xnt.note_id = n.id AND xt.name = '" + exTag + "')"
		}
	}
	err = r.db.QueryRow(countQuery, params[:len(params)-2]...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}
	return notes, total, nil
}

func (r *NotesRepository) getActivitiesForNote(noteID int) ([]models.NoteActivity, error) {
	rows, err := r.db.Query(`
		SELECT a.id, a.type_id, a.value->>'data', at.unit
		FROM activities a
		JOIN activity_types at ON a.type_id = at.id
		WHERE a.note_id = $1
		  AND a.is_deleted = FALSE
	`, noteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.NoteActivity
	for rows.Next() {
		var item models.NoteActivity
		var val string
		if err := rows.Scan(&item.ID, &item.TypeID, &val, &item.Unit); err != nil {
			return nil, err
		}
		item.Value = val
		result = append(result, item)
	}
	return result, nil
}

func joinConditions(conds []string, sep string) string {
	result := ""
	for i, c := range conds {
		if i == 0 {
			result = c
		} else {
			result += sep + c
		}
	}
	return result
}

func truncate(s string, max int) string {
	if len(s) > max {
		return s[:max]
	}
	return s
}

type TagAutocomplete struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func (r *NotesRepository) GetTagAutocomplete(text string, spaceID int, topicID *int) ([]TagAutocomplete, error) {
	query := `
		WITH ranked_tags AS (
			SELECT 
				t.id,
				t.name,
				CASE 
					WHEN tst.topic_id = $3 THEN 0
					ELSE 1
				END as rank
			FROM tag t
			JOIN tag_to_space_topic tst ON t.id = tst.tag_id
			WHERE tst.space_id = $1
			AND ($2 = '' OR t.name ILIKE $2 || '%')
		)
		SELECT id, name
		FROM ranked_tags
		ORDER BY rank, name
		LIMIT 10
	`
	rows, err := r.db.Query(query, spaceID, text, topicID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []TagAutocomplete
	for rows.Next() {
		var tag TagAutocomplete
		if err := rows.Scan(&tag.ID, &tag.Name); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, nil
}
