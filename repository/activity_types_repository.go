package repository

import (
	"database/sql"
	"errors"
	"focuz-api/models"
	"strings"
	"time"
)

type ActivityTypesRepository struct {
	db *sql.DB
}

func NewActivityTypesRepository(db *sql.DB) *ActivityTypesRepository {
	return &ActivityTypesRepository{db: db}
}

func (r *ActivityTypesRepository) CreateActivityType(name, valueType string, minValue, maxValue *float64, aggregation string, spaceID *int, categoryID *int, unit *string) (*models.ActivityType, error) {
	var id int
	now := time.Now()
	err := r.db.QueryRow(`
		INSERT INTO activity_types (name, value_type, min_value, max_value, aggregation, space_id, category_id, unit, created_at, modified_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9)
		RETURNING id
	`, name, valueType, minValue, maxValue, aggregation, spaceID, categoryID, unit, now).Scan(&id)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate key") {
			return nil, errors.New("name conflict in this space")
		}
		return nil, err
	}
	return r.GetActivityTypeByID(id)
}

func (r *ActivityTypesRepository) GetActivityTypeByID(id int) (*models.ActivityType, error) {
	var a models.ActivityType
	var spaceID sql.NullInt64
	var categoryID sql.NullInt64
	var minVal sql.NullFloat64
	var maxVal sql.NullFloat64
	var unit sql.NullString
	err := r.db.QueryRow(`
		SELECT id, name, value_type, min_value, max_value, aggregation, space_id, is_default, is_deleted, unit, category_id, created_at, modified_at
		FROM activity_types
		WHERE id = $1
	`, id).Scan(
		&a.ID,
		&a.Name,
		&a.ValueType,
		&minVal,
		&maxVal,
		&a.Aggregation,
		&spaceID,
		&a.IsDefault,
		&a.IsDeleted,
		&unit,
		&categoryID,
		&a.CreatedAt,
		&a.ModifiedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if spaceID.Valid {
		a.SpaceID = int(spaceID.Int64)
	}
	if categoryID.Valid {
		a.CategoryID = int(categoryID.Int64)
	}
	if minVal.Valid {
		f := minVal.Float64
		a.MinValue = &f
	}
	if maxVal.Valid {
		f := maxVal.Float64
		a.MaxValue = &f
	}
	if unit.Valid {
		u := unit.String
		a.Unit = &u
	}
	return &a, nil
}

func (r *ActivityTypesRepository) UpdateActivityTypeDeleted(id int, isDeleted bool) error {
	_, err := r.db.Exec(`
		UPDATE activity_types
		SET is_deleted = $1, modified_at = NOW()
		WHERE id = $2
	`, isDeleted, id)
	return err
}
