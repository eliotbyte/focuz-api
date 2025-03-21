package models

import "time"

type User struct {
	ID           int       `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"` // Не возвращаем в JSON
	CreatedAt    time.Time `json:"createdAt"`
}
