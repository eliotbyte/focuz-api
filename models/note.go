package models

import "time"

type Note struct {
	ID         int         `json:"id"`
	UserID     int         `json:"userId"`
	Text       string      `json:"text"`
	Tags       []string    `json:"tags"`
	CreatedAt  time.Time   `json:"createdAt"`
	ModifiedAt time.Time   `json:"modifiedAt"`
	Date       time.Time   `json:"date"`
	Parent     *ParentNote `json:"parent"`
	ReplyCount int         `json:"replyCount"`
	IsDeleted  bool        `json:"-"`
	TopicID    int         `json:"topicId"`
}

type ParentNote struct {
	ID   int    `json:"id"`
	Text string `json:"text"`
}
