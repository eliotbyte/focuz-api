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
	err := r.db.QueryRow(`
		INSERT INTO topic (space_id, name, type_id, created_at, is_deleted)
		VALUES ($1, $2, $3, NOW(), FALSE)
		RETURNING id
	`, spaceID, name, typeID).Scan(&spaceID)
	if err != nil {
		return nil, err
	}
	return r.GetTopicByID(spaceID)
}

func (r *TopicsRepository) GetTopicByID(id int) (*models.Topic, error) {
	var t models.Topic
	var typeName string
	err := r.db.QueryRow(`
		SELECT tp.id, tp.space_id, tp.name, tp.type_id, tt.name as type_name, tp.is_deleted, tp.created_at
		FROM topic tp
		INNER JOIN topic_type tt ON tp.type_id = tt.id
		WHERE tp.id = $1
	`, id).Scan(&t.ID, &t.SpaceID, &t.Name, &t.TypeID, &typeName, &t.IsDeleted, &t.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	t.TypeName = typeName
	return &t, nil
}

func (r *TopicsRepository) UpdateTopic(id int, name string, typeID int) error {
	_, err := r.db.Exec(`
		UPDATE topic
		SET name = $1, type_id = $2
		WHERE id = $3
	`, name, typeID, id)
	return err
}

func (r *TopicsRepository) SetTopicDeleted(id int, isDeleted bool) error {
	_, err := r.db.Exec(`
		UPDATE topic
		SET is_deleted = $1
		WHERE id = $2
	`, isDeleted, id)
	return err
}
