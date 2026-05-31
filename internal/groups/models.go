// Package groups enthält Gruppen (Kreise), Mitglieder und Einladungen (Phase 2).
package groups

import "time"

// Group entspricht der Tabelle groups (§6).
type Group struct {
	ID         int64
	Name       string
	OwnerID    int64
	MaxMembers int
	CreatedAt  time.Time
}

// Member entspricht der Tabelle group_members (§6).
type Member struct {
	GroupID   int64
	UserID    int64
	Role      string // owner | member
	InvitedBy *int64
	JoinedAt  time.Time
}

// GroupDetails ist die Detailsicht inkl. Rolle und Mitgliederzahl.
type GroupDetails struct {
	Group       Group
	MyRole      string
	MemberCount int
}
