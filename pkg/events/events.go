package events

// InvitationCreated represents an event emitted when a user is invited to a space.
// This struct is intentionally small and versionable; changes should be additive.
type InvitationCreated struct {
	Type    string `json:"type"`
	SpaceID int    `json:"spaceId"`
	Inviter int    `json:"inviterId"`
}
