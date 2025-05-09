package repository

import (
	"database/sql"
	"focuz-api/models"
	"time"
)

type SpacesRepository struct {
	db *sql.DB
}

type SpaceParticipant struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"createdAt"`
	RoleID    int       `json:"roleId"`
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
		INSERT INTO space (name, owner_id, created_at, modified_at, is_deleted)
		VALUES ($1, $2, NOW(), NOW(), FALSE)
		RETURNING id
	`, name, ownerID).Scan(&spaceID)
	if err != nil {
		return nil, err
	}

	// The role ID of the "owner" is passed in as ownerID here.
	_, err = tx.Exec(`
		INSERT INTO user_to_space (user_id, space_id, role_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, space_id) DO NOTHING
	`, ownerID, spaceID, ownerID)
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
		SELECT id, name, owner_id, is_deleted, created_at, modified_at
		FROM space
		WHERE id = $1
	`, id).Scan(&s.ID, &s.Name, &s.OwnerID, &s.IsDeleted, &s.CreatedAt, &s.ModifiedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// Returns the role_id for the user in the space, or 0 if the user is not a member.
func (r *SpacesRepository) GetUserRoleIDInSpace(userID, spaceID int) (int, error) {
	var roleID int
	err := r.db.QueryRow(`
		SELECT role_id
		FROM user_to_space
		WHERE user_id = $1 AND space_id = $2
	`, userID, spaceID).Scan(&roleID)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return roleID, nil
}

// GetSpacesForUser returns all non-deleted spaces the user belongs to.
func (r *SpacesRepository) GetSpacesForUser(userID int) ([]models.Space, error) {
	rows, err := r.db.Query(`
		SELECT s.id, s.name, s.owner_id, s.is_deleted, s.created_at, s.modified_at
		FROM space s
		INNER JOIN user_to_space uts ON s.id = uts.space_id
		WHERE uts.user_id = $1
		  AND s.is_deleted = FALSE
		ORDER BY s.id
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.Space
	for rows.Next() {
		var s models.Space
		err = rows.Scan(
			&s.ID,
			&s.Name,
			&s.OwnerID,
			&s.IsDeleted,
			&s.CreatedAt,
			&s.ModifiedAt,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	return result, nil
}

func (r *SpacesRepository) InviteUserToSpace(userID, spaceID, roleID int) error {
	_, err := r.db.Exec(`
		INSERT INTO user_to_space (user_id, space_id, role_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, space_id) DO UPDATE SET role_id = EXCLUDED.role_id
	`, userID, spaceID, roleID)
	return err
}

func (r *SpacesRepository) RemoveUserFromSpace(userID, spaceID int) error {
	_, err := r.db.Exec(`
		DELETE FROM user_to_space
		WHERE user_id = $1 AND space_id = $2
	`, userID, spaceID)
	return err
}

func (r *SpacesRepository) GetUsersInSpace(spaceID int) ([]SpaceParticipant, error) {
	rows, err := r.db.Query(`
		SELECT u.id, u.username, u.created_at, uts.role_id
		FROM users u
		INNER JOIN user_to_space uts ON u.id = uts.user_id
		WHERE uts.space_id = $1
	`, spaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var participants []SpaceParticipant
	for rows.Next() {
		var p SpaceParticipant
		err = rows.Scan(&p.ID, &p.Username, &p.CreatedAt, &p.RoleID)
		if err != nil {
			return nil, err
		}
		participants = append(participants, p)
	}
	return participants, nil
}

// Sets or unsets the is_deleted flag on a space.
func (r *SpacesRepository) SetSpaceDeleted(spaceID int, isDeleted bool) error {
	_, err := r.db.Exec(`
		UPDATE space
		SET is_deleted = $1, modified_at = NOW()
		WHERE id = $2
	`, isDeleted, spaceID)
	return err
}

// Updates the name of a space.
func (r *SpacesRepository) UpdateSpaceName(spaceID int, name string) error {
	_, err := r.db.Exec(`
		UPDATE space
		SET name = $1, modified_at = NOW()
		WHERE id = $2
	`, name, spaceID)
	return err
}
