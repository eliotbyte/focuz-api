package repository

import (
	"database/sql"
	"focuz-api/models"
	"time"
)

type ActivitiesRepository struct {
	db *sql.DB
}

func NewActivitiesRepository(db *sql.DB) *ActivitiesRepository {
	return &ActivitiesRepository{db: db}
}

func (r *ActivitiesRepository) CreateActivity(userID, typeID int, value []byte, noteID *int) (*models.Activity, error) {
	var newID int
	now := time.Now()
	err := r.db.QueryRow(`
		INSERT INTO activities (user_id, type_id, value, note_id, created_at, modified_at, is_deleted)
		VALUES ($1, $2, $3, $4, $5, $5, FALSE)
		RETURNING id
	`, userID, typeID, value, noteID, now).Scan(&newID)
	if err != nil {
		return nil, err
	}
	return r.GetActivityByID(newID)
}

func (r *ActivitiesRepository) GetActivityByID(id int) (*models.Activity, error) {
	var a models.Activity
	var rawValue []byte
	var dbNoteID sql.NullInt64
	err := r.db.QueryRow(`
		SELECT id, user_id, type_id, value, note_id, is_deleted, created_at, modified_at
		FROM activities
		WHERE id = $1
	`, id).Scan(
		&a.ID,
		&a.UserID,
		&a.TypeID,
		&rawValue,
		&dbNoteID,
		&a.IsDeleted,
		&a.CreatedAt,
		&a.ModifiedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if dbNoteID.Valid {
		nid := int(dbNoteID.Int64)
		a.NoteID = &nid
	}
	a.Value = rawValue
	return &a, nil
}

func (r *ActivitiesRepository) UpdateActivity(id int, newValue []byte, newNoteID *int) error {
	_, err := r.db.Exec(`
		UPDATE activities
		SET value = $1,
		    note_id = $2,
		    modified_at = NOW()
		WHERE id = $3
	`, newValue, newNoteID, id)
	return err
}

func (r *ActivitiesRepository) SetActivityDeleted(id int, isDeleted bool) error {
	_, err := r.db.Exec(`
		UPDATE activities
		SET is_deleted = $1,
		    modified_at = NOW()
		WHERE id = $2
	`, isDeleted, id)
	return err
}
