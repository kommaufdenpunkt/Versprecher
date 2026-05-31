package posts

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func errorStatus(err error) int {
	switch {
	case errors.Is(err, ErrInvalidInput):
		return http.StatusBadRequest
	case errors.Is(err, ErrNotMember), errors.Is(err, ErrForbidden), errors.Is(err, ErrTaggedNotMember):
		return http.StatusForbidden
	case errors.Is(err, ErrPostNotFound):
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}

func fail(c *gin.Context, err error) {
	status := errorStatus(err)
	msg := err.Error()
	if status == http.StatusInternalServerError {
		msg = "interner Fehler"
	}
	c.JSON(status, gin.H{"error": msg})
}

// postView ist die öffentliche Sicht auf einen Beitrag. ai_score (Moderationswert)
// wird bewusst NICHT an normale Mitglieder ausgegeben.
func postView(p *Post) gin.H {
	return gin.H{
		"id":                  p.ID,
		"group_id":            p.GroupID,
		"author_id":           p.AuthorID,
		"tagged_user_id":      p.TaggedUserID,
		"word":                p.Word,
		"explanation":         p.Explanation,
		"kind":                p.Kind,
		"ai_kind_suggestion":  p.AIKindSuggestion,
		"ai_meant_suggestion": p.AIMeantSuggestion,
		"meant_confirmed":     p.MeantConfirmed,
		"status":              p.Status,
		"is_pinned":           p.IsPinned,
		"created_at":          p.CreatedAt,
	}
}

type createRequest struct {
	TaggedUserID *int64 `json:"tagged_user_id"`
	Word         string `json:"word"`
	Explanation  string `json:"explanation"`
	Kind         string `json:"kind"`
	VoiceURL     string `json:"voice_url"`
}

// Create: POST /v1/groups/:id/posts
func (h *Handler) Create(c *gin.Context) {
	groupID, ok := parseID(c, "id")
	if !ok {
		return
	}
	var req createRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, ErrInvalidInput)
		return
	}
	p, err := h.svc.Create(c.Request.Context(), CreateInput{
		GroupID:      groupID,
		AuthorID:     c.GetInt64("uid"),
		TaggedUserID: req.TaggedUserID,
		Word:         req.Word,
		Explanation:  req.Explanation,
		Kind:         req.Kind,
		VoiceURL:     req.VoiceURL,
	})
	if err != nil {
		fail(c, err)
		return
	}
	// 202: angenommen, wird von Fidolin geprüft (noch nicht im Feed sichtbar).
	c.JSON(http.StatusAccepted, gin.H{
		"post":    postView(p),
		"hinweis": "Beitrag wird geprüft und erscheint nach der Freigabe im Feed.",
	})
}

// Feed: GET /v1/groups/:id/feed?limit=&before=
func (h *Handler) Feed(c *gin.Context) {
	groupID, ok := parseID(c, "id")
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(c.Query("limit"))
	before, _ := strconv.ParseInt(c.DefaultQuery("before", "0"), 10, 64)

	list, err := h.svc.Feed(c.Request.Context(), c.GetInt64("uid"), groupID, before, limit)
	if err != nil {
		fail(c, err)
		return
	}
	out := make([]gin.H, 0, len(list))
	var nextBefore int64
	for i := range list {
		out = append(out, postView(&list[i]))
		nextBefore = list[i].ID
	}
	c.JSON(http.StatusOK, gin.H{"posts": out, "next_before": nextBefore})
}

type confirmRequest struct {
	Meant string `json:"meant"`
	Kind  string `json:"kind"`
}

// ConfirmMeant: PATCH /v1/posts/:id
func (h *Handler) ConfirmMeant(c *gin.Context) {
	postID, ok := parseID(c, "id")
	if !ok {
		return
	}
	var req confirmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, ErrInvalidInput)
		return
	}
	p, err := h.svc.ConfirmMeant(c.Request.Context(), c.GetInt64("uid"), postID, req.Meant, req.Kind)
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"post": postView(p)})
}

func parseID(c *gin.Context, name string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || id <= 0 {
		fail(c, ErrInvalidInput)
		return 0, false
	}
	return id, true
}
