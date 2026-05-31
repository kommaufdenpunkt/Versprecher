// Package posts enthält Beiträge (Versprecher/Verhörer) und den Feed (Phase 3).
package posts

import "time"

// Post entspricht der Tabelle posts (§6) — ohne die Fidolin-internen Felder
// (ai_status/ai_claimed_at), die das fidolin-Paket verwaltet.
type Post struct {
	ID                int64
	GroupID           int64
	AuthorID          int64
	TaggedUserID      *int64
	Word              string
	WordNormalized    string
	Explanation       string
	Kind              *string
	AIKindSuggestion  *string
	AIMeantSuggestion *string
	MeantConfirmed    *string
	VoiceURL          *string
	AIScore           *float64
	Status            string
	IsPinned          bool
	CreatedAt         time.Time
}

// CreateParams bündelt die Felder zum Anlegen eines Beitrags.
type CreateParams struct {
	GroupID        int64
	AuthorID       int64
	TaggedUserID   *int64
	Word           string
	WordNormalized string
	Explanation    string
	Kind           *string
	VoiceURL       *string
}
