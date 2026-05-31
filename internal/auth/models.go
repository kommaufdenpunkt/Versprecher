package auth

import "time"

// User entspricht der Tabelle users (§6).
type User struct {
	ID              int64
	Email           string
	EmailVerifiedAt *time.Time
	PasswordHash    string
	DisplayName     string
	Role            string
	Status          string
	InvitedBy       *int64
	CreatedAt       time.Time
}

func (u *User) IsAdmin() bool       { return u.Role == "admin" }
func (u *User) EmailVerified() bool { return u.EmailVerifiedAt != nil }

// Invitation entspricht der Tabelle invitations (§6). Token ist der sha256-Hash.
type Invitation struct {
	ID        int64
	GroupID   *int64
	InviterID int64
	Token     string
	Email     *string
	Status    string
	ExpiresAt *time.Time
	CreatedAt time.Time
}

// EmailVerification entspricht der Tabelle email_verifications.
type EmailVerification struct {
	ID        int64
	UserID    int64
	Token     string
	ExpiresAt time.Time
}

// CreateUserParams bündelt die Felder zum Anlegen eines Nutzers.
type CreateUserParams struct {
	Email        string
	PasswordHash string
	DisplayName  string
	Role         string
	InvitedBy    *int64
}
