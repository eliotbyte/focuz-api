package repository

import (
	"database/sql"
)

type Notification struct {
	ID      int
	UserID  int
	Type    string
	Payload []byte
	IsRead  bool
	Sticky  bool
}

type NotificationsRepository struct {
	db *sql.DB
}

func NewNotificationsRepository(db *sql.DB) *NotificationsRepository {
	return &NotificationsRepository{db: db}
}

func (r *NotificationsRepository) Create(userID int, notifType string, payload []byte, sticky bool) error {
	_, err := r.db.Exec(`
		INSERT INTO notifications (user_id, type, payload, sticky)
		VALUES ($1, $2, $3, $4)
	`, userID, notifType, payload, sticky)
	return err
}

func (r *NotificationsRepository) ListUnread(userID int) ([]Notification, error) {
	rows, err := r.db.Query(`
		SELECT id, user_id, type, payload, is_read, sticky
		FROM notifications
		WHERE user_id = $1 AND is_read = FALSE
		ORDER BY sticky DESC, created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []Notification
	for rows.Next() {
		n := Notification{}
		var payload []byte
		if err := rows.Scan(&n.ID, &n.UserID, &n.Type, &payload, &n.IsRead, &n.Sticky); err != nil {
			return nil, err
		}
		n.Payload = payload
		result = append(result, n)
	}
	return result, nil
}

func (r *NotificationsRepository) MarkRead(userID int, ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	// Simple approach: update in a loop to avoid SQL array binding complexity here
	for _, id := range ids {
		_, err := r.db.Exec(`
			UPDATE notifications SET is_read = TRUE, read_at = NOW()
			WHERE id = $1 AND user_id = $2
		`, id, userID)
		if err != nil {
			return err
		}
	}
	return nil
}
