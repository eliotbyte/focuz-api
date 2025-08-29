package repository

import (
	"database/sql"
	"encoding/json"
	"focuz-api/models"
	"focuz-api/types"
	"time"

	"github.com/lib/pq"
)

type SyncRepository struct {
	db *sql.DB
}

func NewSyncRepository(db *sql.DB) *SyncRepository { return &SyncRepository{db: db} }

func (r *SyncRepository) GetChangesSince(userID int, accessibleSpaceIDs []int, since time.Time) (*types.SyncPullResponse, error) {
	resp := &types.SyncPullResponse{}

	// Spaces
	rows, err := r.db.Query(`
		SELECT id, name, created_at, modified_at, is_deleted
		FROM space
		WHERE id = ANY($1)
		AND modified_at > $2
	`, pq.Array(accessibleSpaceIDs), since)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var id int
		var name string
		var created, modified time.Time
		var isDeleted bool
		if err := rows.Scan(&id, &name, &created, &modified, &isDeleted); err != nil {
			rows.Close()
			return nil, err
		}
		var deletedAt *time.Time
		if isDeleted {
			deletedAt = &modified
		}
		resp.Spaces = append(resp.Spaces, types.SpaceChange{ID: id, Name: name, CreatedAt: created, ModifiedAt: modified, DeletedAt: deletedAt})
	}
	rows.Close()

	// Notes (include deleted)
	noteRows, err := r.db.Query(`
		SELECT n.id, n.user_id, n.space_id, n.text, n.date, n.parent_id, n.created_at, n.modified_at, n.is_deleted,
		  COALESCE((SELECT ARRAY_AGG(t.name ORDER BY t.name)
		           FROM note_to_tag nt JOIN tag t ON t.id = nt.tag_id
		           WHERE nt.note_id = n.id), ARRAY[]::text[]) AS tags
		FROM note n
		WHERE n.space_id = ANY($1)
		AND n.modified_at > $2
		ORDER BY n.id
	`, pq.Array(accessibleSpaceIDs), since)
	if err != nil {
		return nil, err
	}
	for noteRows.Next() {
		var id, userIDRow, spaceID int
		var text string
		var created, modified time.Time
		var isDeleted bool
		var date time.Time
		var parentID sql.NullInt64
		var tags []string
		if err := noteRows.Scan(&id, &userIDRow, &spaceID, &text, &date, &parentID, &created, &modified, &isDeleted, pq.Array(&tags)); err != nil {
			noteRows.Close()
			return nil, err
		}
		var deletedAt *time.Time
		if isDeleted {
			deletedAt = &modified
		}
		var datePtr *time.Time = &date
		if date.IsZero() {
			datePtr = nil
		}
		var parentPtr *int
		if parentID.Valid {
			tmp := int(parentID.Int64)
			parentPtr = &tmp
		}
		resp.Notes = append(resp.Notes, types.NoteChange{
			ID:         &id,
			SpaceID:    spaceID,
			UserID:     userIDRow,
			Text:       &text,
			Tags:       tags,
			Date:       datePtr,
			ParentID:   parentPtr,
			CreatedAt:  created,
			ModifiedAt: modified,
			DeletedAt:  deletedAt,
		})
	}
	noteRows.Close()

	// Tags per space (no delete tracking yet)
	tagRows, err := r.db.Query(`
		SELECT t.id, ts.space_id, t.name, ts.created_at
		FROM tag t JOIN tag_to_space ts ON ts.tag_id = t.id
		WHERE ts.space_id = ANY($1)
		AND ts.created_at > $2
		ORDER BY ts.space_id, t.name
	`, pq.Array(accessibleSpaceIDs), since)
	if err != nil {
		return nil, err
	}
	for tagRows.Next() {
		var id, spaceID int
		var name string
		var created time.Time
		if err := tagRows.Scan(&id, &spaceID, &name, &created); err != nil {
			tagRows.Close()
			return nil, err
		}
		resp.Tags = append(resp.Tags, types.TagChange{ID: id, SpaceID: spaceID, Name: name, CreatedAt: created, ModifiedAt: created})
	}
	tagRows.Close()

	// Filters
	filterRows, err := r.db.Query(`
		SELECT id, user_id, space_id, parent_id, name, params, created_at, modified_at, is_deleted
		FROM filters
		WHERE space_id = ANY($1)
		AND modified_at > $2
	`, pq.Array(accessibleSpaceIDs), since)
	if err != nil {
		return nil, err
	}
	for filterRows.Next() {
		var id, userIDRow, spaceID int
		var name string
		var paramsRaw []byte
		var created, modified time.Time
		var isDeleted bool
		var parentID sql.NullInt64
		if err := filterRows.Scan(&id, &userIDRow, &spaceID, &parentID, &name, &paramsRaw, &created, &modified, &isDeleted); err != nil {
			filterRows.Close()
			return nil, err
		}
		var params interface{}
		_ = json.Unmarshal(paramsRaw, &params)
		var deletedAt *time.Time
		if isDeleted {
			deletedAt = &modified
		}
		var parentPtr *int
		if parentID.Valid {
			tmp := int(parentID.Int64)
			parentPtr = &tmp
		}
		resp.Filters = append(resp.Filters, types.FilterChange{ID: id, SpaceID: spaceID, UserID: userIDRow, ParentID: parentPtr, Name: name, Params: params, CreatedAt: created, ModifiedAt: modified, DeletedAt: deletedAt})
	}
	filterRows.Close()

	// Charts
	chartRows, err := r.db.Query(`
		SELECT id, user_id, space_id, kind, activity_type_id, period, name, description, note_id, created_at, modified_at, is_deleted
		FROM chart
		WHERE space_id = ANY($1)
		AND modified_at > $2
	`, pq.Array(accessibleSpaceIDs), since)
	if err != nil {
		return nil, err
	}
	for chartRows.Next() {
		var it types.ChartChange
		var isDeleted bool
		if err := chartRows.Scan(&it.ID, &it.UserID, &it.SpaceID, &it.KindID, &it.ActivityTypeID, &it.PeriodID, &it.Name, &it.Description, &it.NoteID, &it.CreatedAt, &it.ModifiedAt, &isDeleted); err != nil {
			chartRows.Close()
			return nil, err
		}
		if isDeleted {
			it.DeletedAt = &it.ModifiedAt
		}
		resp.Charts = append(resp.Charts, it)
	}
	chartRows.Close()

	// Activities
	actRows, err := r.db.Query(`
		SELECT id, user_id, type_id, value, note_id, created_at, modified_at, is_deleted
		FROM activities
		WHERE modified_at > $2
		AND (note_id IS NULL OR EXISTS (SELECT 1 FROM note n WHERE n.id = activities.note_id AND n.space_id = ANY($1)))
	`, pq.Array(accessibleSpaceIDs), since)
	if err != nil {
		return nil, err
	}
	for actRows.Next() {
		var it types.ActivityChange
		var raw []byte
		var isDeleted bool
		if err := actRows.Scan(&it.ID, &it.UserID, &it.TypeID, &raw, &it.NoteID, &it.CreatedAt, &it.ModifiedAt, &isDeleted); err != nil {
			actRows.Close()
			return nil, err
		}
		var anyVal interface{}
		_ = json.Unmarshal(raw, &anyVal)
		it.Value = anyVal
		if isDeleted {
			it.DeletedAt = &it.ModifiedAt
		}
		resp.Activities = append(resp.Activities, it)
	}
	actRows.Close()

	// Attachments for notes in accessible spaces
	attRows, err := r.db.Query(`
		SELECT a.id, a.note_id, a.file_name, a.file_type, a.file_size, a.created_at, a.modified_at
		FROM attachments a
		WHERE a.modified_at > $2
		AND EXISTS (SELECT 1 FROM note n WHERE n.id = a.note_id AND n.space_id = ANY($1))
	`, pq.Array(accessibleSpaceIDs), since)
	if err != nil {
		return nil, err
	}
	for attRows.Next() {
		var it types.AttachmentChange
		if err := attRows.Scan(&it.ID, &it.NoteID, &it.FileName, &it.FileType, &it.FileSize, &it.CreatedAt, &it.ModifiedAt); err != nil {
			attRows.Close()
			return nil, err
		}
		resp.Attachments = append(resp.Attachments, it)
	}
	attRows.Close()

	// Activity types (default or space-specific)
	atyRows, err := r.db.Query(`
		SELECT id, name, value_type, min_value, max_value, aggregation, space_id, is_default, unit, category_id, created_at, modified_at
		FROM activity_types
		WHERE modified_at > $2
		AND (is_default = TRUE OR space_id = ANY($1))
		AND is_deleted = FALSE
	`, pq.Array(accessibleSpaceIDs), since)
	if err != nil {
		return nil, err
	}
	for atyRows.Next() {
		var it types.ActivityTypeChange
		var spaceID sql.NullInt64
		var minV, maxV sql.NullFloat64
		var unit sql.NullString
		var catID sql.NullInt64
		if err := atyRows.Scan(&it.ID, &it.Name, &it.ValueType, &minV, &maxV, &it.Aggregation, &spaceID, &it.IsDefault, &unit, &catID, &it.CreatedAt, &it.ModifiedAt); err != nil {
			atyRows.Close()
			return nil, err
		}
		if spaceID.Valid {
			tmp := int(spaceID.Int64)
			it.SpaceID = &tmp
		}
		if minV.Valid {
			tmp := minV.Float64
			it.MinValue = &tmp
		}
		if maxV.Valid {
			tmp := maxV.Float64
			it.MaxValue = &tmp
		}
		if unit.Valid {
			it.Unit = &unit.String
		}
		if catID.Valid {
			tmp := int(catID.Int64)
			it.CategoryID = &tmp
		}
		resp.ActivityTypes = append(resp.ActivityTypes, it)
	}
	atyRows.Close()

	return resp, nil
}

// ApplyChanges applies client changes with last-write-wins policy.
func (r *SyncRepository) ApplyChanges(userID int, payload types.SyncPushRequest) (*types.SyncPushResponse, error) {
	resp := &types.SyncPushResponse{Applied: 0}

	// Notes
	for _, n := range payload.Notes {
		if n.SpaceID == 0 {
			continue
		}
		// Create new when no ID provided
		if n.ID == nil {
			if n.Text == nil {
				continue
			}
			var newID int
			// Insert note
			err := r.db.QueryRow(`
				INSERT INTO note (user_id, text, created_at, modified_at, date, parent_id, space_id, is_deleted)
				VALUES ($1, $2, COALESCE($3, NOW()), COALESCE($4, NOW()), COALESCE($5, NOW()), $6, $7, FALSE)
				RETURNING id
			`, userID, *n.Text, n.CreatedAt, n.ModifiedAt, n.Date, n.ParentID, n.SpaceID).Scan(&newID)
			if err != nil {
				return nil, err
			}
			// Replace tags
			if err := r.replaceNoteTags(newID, n.Tags, n.SpaceID); err != nil {
				return nil, err
			}
			resp.Applied++
			if n.ClientID != nil {
				resp.Mappings = append(resp.Mappings, types.Mapping{Resource: "note", ClientID: *n.ClientID, ServerID: newID})
			}
			continue
		}
		// Update existing with LWW
		var serverModified time.Time
		err := r.db.QueryRow(`SELECT modified_at FROM note WHERE id = $1`, *n.ID).Scan(&serverModified)
		if err == sql.ErrNoRows {
			// Treat as create with forced id
			var newID int
			err := r.db.QueryRow(`
				INSERT INTO note (id, user_id, text, created_at, modified_at, date, parent_id, space_id, is_deleted)
				VALUES ($1, $2, $3, COALESCE($4, NOW()), COALESCE($5, NOW()), COALESCE($6, NOW()), $7, $8, $9)
				RETURNING id
			`, *n.ID, userID, toString(n.Text), n.CreatedAt, n.ModifiedAt, n.Date, n.ParentID, n.SpaceID, n.DeletedAt != nil).Scan(&newID)
			if err != nil {
				return nil, err
			}
			if err := r.replaceNoteTags(newID, n.Tags, n.SpaceID); err != nil {
				return nil, err
			}
			resp.Applied++
			continue
		} else if err != nil {
			return nil, err
		}
		if n.ModifiedAt.After(serverModified) {
			_, err := r.db.Exec(`
				UPDATE note SET text = COALESCE($2, text), date = COALESCE($3, date), parent_id = $4, is_deleted = $5, modified_at = $6 WHERE id = $1
			`, *n.ID, n.Text, n.Date, n.ParentID, n.DeletedAt != nil, n.ModifiedAt)
			if err != nil {
				return nil, err
			}
			if err := r.replaceNoteTags(*n.ID, n.Tags, n.SpaceID); err != nil {
				return nil, err
			}
			resp.Applied++
		} else {
			var current models.Note
			err := r.db.QueryRow(`SELECT id, user_id, space_id, text, created_at, modified_at, is_deleted, date, parent_id FROM note WHERE id = $1`, *n.ID).
				Scan(&current.ID, &current.UserID, &current.SpaceID, &current.Text, &current.CreatedAt, &current.ModifiedAt, &current.IsDeleted, &current.Date, new(sql.NullInt64))
			if err == nil {
				var deletedAt *time.Time
				if current.IsDeleted {
					deletedAt = &current.ModifiedAt
				}
				server := types.NoteChange{ID: &current.ID, SpaceID: current.SpaceID, UserID: current.UserID, Text: &current.Text, Tags: []string{}, Date: &current.Date, CreatedAt: current.CreatedAt, ModifiedAt: current.ModifiedAt, DeletedAt: deletedAt}
				resp.Conflicts = append(resp.Conflicts, types.Conflict{Resource: "note", ID: current.ID, Reason: "server-newer", Server: server})
			}
		}
	}

	// Filters (LWW on name/params/parent)
	for _, f := range payload.Filters {
		if f.ID == 0 {
			continue
		}
		var serverModified time.Time
		err := r.db.QueryRow(`SELECT modified_at FROM filters WHERE id = $1`, f.ID).Scan(&serverModified)
		if err == sql.ErrNoRows {
			paramsBytes, _ := json.Marshal(f.Params)
			_, err := r.db.Exec(`
				INSERT INTO filters (id, user_id, space_id, parent_id, name, params, is_deleted, created_at, modified_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7, COALESCE($8, NOW()), COALESCE($9, NOW()))
			`, f.ID, f.UserID, f.SpaceID, f.ParentID, f.Name, paramsBytes, f.DeletedAt != nil, f.CreatedAt, f.ModifiedAt)
			if err != nil {
				return nil, err
			}
			resp.Applied++
			continue
		} else if err != nil {
			return nil, err
		}
		if f.ModifiedAt.After(serverModified) {
			paramsBytes, _ := json.Marshal(f.Params)
			_, err := r.db.Exec(`UPDATE filters SET name = $2, parent_id = $3, params = $4, is_deleted = $5, modified_at = $6 WHERE id = $1`, f.ID, f.Name, f.ParentID, paramsBytes, f.DeletedAt != nil, f.ModifiedAt)
			if err != nil {
				return nil, err
			}
			resp.Applied++
		} else {
			resp.Conflicts = append(resp.Conflicts, types.Conflict{Resource: "filter", ID: f.ID, Reason: "server-newer"})
		}
	}

	// Charts and Activities unchanged
	for _, ch := range payload.Charts {
		var serverModified time.Time
		err := r.db.QueryRow(`SELECT modified_at FROM chart WHERE id = $1`, ch.ID).Scan(&serverModified)
		if err == sql.ErrNoRows {
			continue
		} else if err != nil {
			return nil, err
		}
		if ch.ModifiedAt.After(serverModified) {
			_, err := r.db.Exec(`UPDATE chart SET name = $2, description = $3, is_deleted = $4, modified_at = $5 WHERE id = $1`, ch.ID, ch.Name, ch.Description, ch.DeletedAt != nil, ch.ModifiedAt)
			if err != nil {
				return nil, err
			}
			resp.Applied++
		} else {
			resp.Conflicts = append(resp.Conflicts, types.Conflict{Resource: "chart", ID: ch.ID, Reason: "server-newer"})
		}
	}
	for _, a := range payload.Activities {
		var serverModified time.Time
		err := r.db.QueryRow(`SELECT modified_at FROM activities WHERE id = $1`, a.ID).Scan(&serverModified)
		if err == sql.ErrNoRows {
			continue
		} else if err != nil {
			return nil, err
		}
		if a.ModifiedAt.After(serverModified) {
			val, _ := json.Marshal(a.Value)
			_, err := r.db.Exec(`UPDATE activities SET value = $2, is_deleted = $3, modified_at = $4 WHERE id = $1`, a.ID, val, a.DeletedAt != nil, a.ModifiedAt)
			if err != nil {
				return nil, err
			}
			resp.Applied++
		} else {
			resp.Conflicts = append(resp.Conflicts, types.Conflict{Resource: "activity", ID: a.ID, Reason: "server-newer"})
		}
	}

	return resp, nil
}

func (r *SyncRepository) replaceNoteTags(noteID int, tags []string, spaceID int) error {
	_, err := r.db.Exec(`DELETE FROM note_to_tag WHERE note_id = $1`, noteID)
	if err != nil {
		return err
	}
	for _, name := range tags {
		var tagID int
		if err := r.db.QueryRow(`INSERT INTO tag (name) VALUES ($1) ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name RETURNING id`, name).Scan(&tagID); err != nil {
			return err
		}
		if _, err := r.db.Exec(`INSERT INTO note_to_tag (note_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, noteID, tagID); err != nil {
			return err
		}
		if _, err := r.db.Exec(`INSERT INTO tag_to_space (tag_id, space_id, created_at) VALUES ($1, $2, NOW()) ON CONFLICT (tag_id, space_id) DO NOTHING`, tagID, spaceID); err != nil {
			return err
		}
	}
	return nil
}

func toString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
