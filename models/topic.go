package models

import "time"

type Topic struct {
	ID         int       `json:"id"`
	SpaceID    int       `json:"spaceId"`
	Name       string    `json:"name"`
	TypeID     int       `json:"typeId"`
	IsDeleted  bool      `json:"-"`
	CreatedAt  time.Time `json:"createdAt"`
	ModifiedAt time.Time `json:"modifiedAt"`
}
