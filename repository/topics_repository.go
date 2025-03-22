package repository

import (
	"database/sql"
	"focuz-api/models"
)

type TopicsRepository struct {
	db *sql.DB
}

func NewTopicsRepository(db *sql.DB) *TopicsRepository {
	return &TopicsRepository{db: db}
}

func (r *TopicsRepository) CreateTopic(spaceID int, name string, typeID int) (*models.Topic, error) {
	var newID int
	err := r.db.QueryRow(`
		INSERT INTO topic (space_id, name, type_id, created_at, modified_at, is_deleted)
		VALUES ($1, $2, $3, NOW(), NOW(), FALSE)
		RETURNING id
	`, spaceID, name, typeID).Scan(&newID)
	if err != nil {
		return nil, err
	}
	return r.GetTopicByID(newID)
}

func (r *TopicsRepository) GetTopicByID(id int) (*models.Topic, error) {
	var t models.Topic
	var typeName string
	err := r.db.QueryRow(`
		SELECT tp.id,
		       tp.space_id,
		       tp.name,
		       tp.type_id,
		       tt.name as type_name,
		       tp.is_deleted,
		       tp.created_at,
		       tp.modified_at
		FROM topic tp
		INNER JOIN topic_type tt ON tp.type_id = tt.id
		WHERE tp.id = $1
	`, id).Scan(
		&t.ID,
		&t.SpaceID,
		&t.Name,
		&t.TypeID,
		&typeName,
		&t.IsDeleted,
		&t.CreatedAt,
		&t.ModifiedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	t.TypeName = typeName
	return &t, nil
}

func (r *TopicsRepository) UpdateTopicName(id int, name string) error {
	_, err := r.db.Exec(`
		UPDATE topic
		SET name = $1, modified_at = NOW()
		WHERE id = $2
	`, name, id)
	return err
}

func (r *TopicsRepository) SetTopicDeleted(id int, isDeleted bool) error {
	_, err := r.db.Exec(`
		UPDATE topic
		SET is_deleted = $1, modified_at = NOW()
		WHERE id = $2
	`, isDeleted, id)
	return err
}

func (r *TopicsRepository) GetTopicsBySpace(spaceID int) ([]*models.Topic, error) {
	rows, err := r.db.Query(`
		SELECT id,
		       space_id,
		       name,
		       type_id,
		       is_deleted,
		       created_at,
		       modified_at
		FROM topic
		WHERE space_id = $1
		  AND is_deleted = FALSE
		ORDER BY id
	`, spaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*models.Topic
	for rows.Next() {
		var t models.Topic
		err = rows.Scan(
			&t.ID,
			&t.SpaceID,
			&t.Name,
			&t.TypeID,
			&t.IsDeleted,
			&t.CreatedAt,
			&t.ModifiedAt,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, &t)
	}
	return result, nil
}
