package types

import "time"

// SyncPullResponse represents all changes since a given timestamp.
type SyncPullResponse struct {
	Spaces        []SpaceChange        `json:"spaces"`
	Notes         []NoteChange         `json:"notes"`
	Tags          []TagChange          `json:"tags"`
	Filters       []FilterChange       `json:"filters"`
	Charts        []ChartChange        `json:"charts"`
	Activities    []ActivityChange     `json:"activities"`
	Attachments   []AttachmentChange   `json:"attachments"`
	ActivityTypes []ActivityTypeChange `json:"activityTypes"`
}

type SpaceChange struct {
	ID         int        `json:"id"`
	Name       string     `json:"name"`
	CreatedAt  time.Time  `json:"created_at"`
	ModifiedAt time.Time  `json:"modified_at"`
	DeletedAt  *time.Time `json:"deleted_at,omitempty"`
}

type NoteChange struct {
	ID         *int       `json:"id,omitempty"`
	ClientID   *string    `json:"clientId,omitempty"`
	SpaceID    int        `json:"space_id"`
	UserID     int        `json:"user_id"`
	Text       *string    `json:"text,omitempty"`
	Tags       []string   `json:"tags"`
	Date       *time.Time `json:"date,omitempty"`
	ParentID   *int       `json:"parent_id,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	ModifiedAt time.Time  `json:"modified_at"`
	DeletedAt  *time.Time `json:"deleted_at,omitempty"`
	// Attachments are included for pull; for push, clients may include only id, file_name, modified_at, is_deleted
	Attachments []AttachmentChange `json:"attachments,omitempty"`
}

type TagChange struct {
	ID         int        `json:"id"`
	SpaceID    int        `json:"space_id"`
	Name       string     `json:"name"`
	CreatedAt  time.Time  `json:"created_at"`
	ModifiedAt time.Time  `json:"modified_at"`
	DeletedAt  *time.Time `json:"deleted_at,omitempty"`
}

type FilterChange struct {
	ID         int         `json:"id"`
	SpaceID    int         `json:"space_id"`
	UserID     int         `json:"user_id"`
	ParentID   *int        `json:"parent_id,omitempty"`
	Params     interface{} `json:"params"`
	Name       string      `json:"name"`
	CreatedAt  time.Time   `json:"created_at"`
	ModifiedAt time.Time   `json:"modified_at"`
	DeletedAt  *time.Time  `json:"deleted_at,omitempty"`
}

type ChartChange struct {
	ID             int        `json:"id"`
	SpaceID        int        `json:"space_id"`
	UserID         int        `json:"user_id"`
	KindID         int        `json:"kind_id"`
	ActivityTypeID int        `json:"activity_type_id"`
	PeriodID       int        `json:"period_id"`
	Name           string     `json:"name"`
	Description    *string    `json:"description,omitempty"`
	NoteID         *int       `json:"note_id,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	ModifiedAt     time.Time  `json:"modified_at"`
	DeletedAt      *time.Time `json:"deleted_at,omitempty"`
}

type ActivityChange struct {
	ID         int         `json:"id"`
	UserID     int         `json:"user_id"`
	NoteID     *int        `json:"note_id,omitempty"`
	TypeID     int         `json:"type_id"`
	Value      interface{} `json:"value"`
	CreatedAt  time.Time   `json:"created_at"`
	ModifiedAt time.Time   `json:"modified_at"`
	DeletedAt  *time.Time  `json:"deleted_at,omitempty"`
}

type AttachmentChange struct {
	ID         string    `json:"id"`
	NoteID     int       `json:"note_id"`
	FileName   string    `json:"file_name"`
	FileType   string    `json:"file_type"`
	FileSize   int64     `json:"file_size"`
	CreatedAt  time.Time `json:"created_at"`
	ModifiedAt time.Time `json:"modified_at"`
	// For push only. Server will ignore for pull.
	IsDeleted *bool `json:"is_deleted,omitempty"`
}

type ActivityTypeChange struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	ValueType   string    `json:"value_type"`
	MinValue    *float64  `json:"min_value,omitempty"`
	MaxValue    *float64  `json:"max_value,omitempty"`
	Aggregation string    `json:"aggregation"`
	SpaceID     *int      `json:"space_id,omitempty"`
	IsDefault   bool      `json:"is_default"`
	Unit        *string   `json:"unit,omitempty"`
	CategoryID  *int      `json:"category_id,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	ModifiedAt  time.Time `json:"modified_at"`
}

// SyncPushRequest contains local changes from client.
type SyncPushRequest struct {
	Notes      []NoteChange     `json:"notes"`
	Tags       []TagChange      `json:"tags"`
	Filters    []FilterChange   `json:"filters"`
	Charts     []ChartChange    `json:"charts"`
	Activities []ActivityChange `json:"activities"`
}

// Conflict describes a resource-level conflict returned to client.
type Conflict struct {
	Resource string      `json:"resource"`
	ID       int         `json:"id"`
	Reason   string      `json:"reason"`
	Server   interface{} `json:"server"`
}

// Mapping returns server IDs for client-generated temporary identifiers.
type Mapping struct {
	Resource string `json:"resource"`
	ClientID string `json:"clientId"`
	ServerID int    `json:"serverId"`
}

// SyncPushResponse acknowledges applied changes and conflicts.
type SyncPushResponse struct {
	Applied   int        `json:"applied"`
	Conflicts []Conflict `json:"conflicts"`
	Mappings  []Mapping  `json:"mappings"`
}
