package repository

import (
	"database/sql"
	"focuz-api/models"
)

type SpacesRepository struct {
	db *sql.DB
}

func NewSpacesRepository(db *sql.DB) *SpacesRepository {
	return &SpacesRepository{db: db}
}

func (r *SpacesRepository) CreateSpace(name string, ownerID int) (*models.Space, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var spaceID int
	err = tx.QueryRow(`
		INSERT INTO space (name, owner_id, created_at, is_deleted)
		VALUES ($1, $2, NOW(), FALSE)
		RETURNING id
	`, name, ownerID).Scan(&spaceID)
	if err != nil {
		return nil, err
	}
	var roleID int
	err = tx.QueryRow(`
		SELECT id FROM role WHERE name = 'owner'
	`).Scan(&roleID)
	if err != nil {
		return nil, err
	}
	_, err = tx.Exec(`
		INSERT INTO user_to_space (user_id, space_id, role_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, space_id) DO NOTHING
	`, ownerID, spaceID, roleID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return r.GetSpaceByID(spaceID)
}

func (r *SpacesRepository) GetSpaceByID(id int) (*models.Space, error) {
	var s models.Space
	err := r.db.QueryRow(`
		SELECT id, name, owner_id, is_deleted, created_at
		FROM space
		WHERE id = $1
	`, id).Scan(&s.ID, &s.Name, &s.OwnerID, &s.IsDeleted, &s.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *SpacesRepository) CanUserEditSpace(userID, spaceID int) (bool, error) {
	var count int
	err := r.db.QueryRow(`
		SELECT COUNT(*)
		FROM user_to_space uts
		INNER JOIN role r ON uts.role_id = r.id
		WHERE uts.user_id = $1
		  AND uts.space_id = $2
		  AND r.name = 'owner'
	`, userID, spaceID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *SpacesRepository) UpdateSpaceName(spaceID int, name string) error {
	_, err := r.db.Exec(`
		UPDATE space
		SET name = $1
		WHERE id = $2
	`, name, spaceID)
	return err
}

func (r *SpacesRepository) SetSpaceDeleted(spaceID int, isDeleted bool) error {
	_, err := r.db.Exec(`
		UPDATE space
		SET is_deleted = $1
		WHERE id = $2
	`, isDeleted, spaceID)
	return err
}

func (r *SpacesRepository) UserHasAccessToSpace(userID, spaceID int) (bool, string, error) {
	var roleName string
	err := r.db.QueryRow(`
		SELECT r.name
		FROM user_to_space uts
		INNER JOIN role r ON uts.role_id = r.id
		WHERE uts.user_id = $1
		  AND uts.space_id = $2
	`, userID, spaceID).Scan(&roleName)
	if err == sql.ErrNoRows {
		return false, "", nil
	}
	if err != nil {
		return false, "", err
	}
	return true, roleName, nil
}

func (r *SpacesRepository) InviteUserToSpace(userID, spaceID, roleID int) error {
	_, err := r.db.Exec(`
		INSERT INTO user_to_space (user_id, space_id, role_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, space_id) DO UPDATE SET role_id = EXCLUDED.role_id
	`, userID, spaceID, roleID)
	return err
}
