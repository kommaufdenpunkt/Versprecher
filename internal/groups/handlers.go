package groups

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Handler bündelt die HTTP-Endpunkte für Gruppen.
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func errorStatus(err error) int {
	switch {
	case errors.Is(err, ErrInvalidInput):
		return http.StatusBadRequest
	case errors.Is(err, ErrNotMember):
		return http.StatusForbidden
	case errors.Is(err, ErrGroupNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrAlreadyMember), errors.Is(err, ErrGroupFull):
		return http.StatusConflict
	case errors.Is(err, ErrInviteInvalid):
		return http.StatusForbidden
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

func groupView(g *Group) gin.H {
	return gin.H{
		"id":          g.ID,
		"name":        g.Name,
		"owner_id":    g.OwnerID,
		"max_members": g.MaxMembers,
		"created_at":  g.CreatedAt,
	}
}

type createGroupRequest struct {
	Name string `json:"name"`
}

// CreateGroup: POST /v1/groups
func (h *Handler) CreateGroup(c *gin.Context) {
	var req createGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, ErrInvalidInput)
		return
	}
	g, err := h.svc.CreateGroup(c.Request.Context(), c.GetInt64("uid"), req.Name)
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"group": groupView(g)})
}

// ListMyGroups: GET /v1/groups
func (h *Handler) ListMyGroups(c *gin.Context) {
	gs, err := h.svc.ListMyGroups(c.Request.Context(), c.GetInt64("uid"))
	if err != nil {
		fail(c, err)
		return
	}
	out := make([]gin.H, 0, len(gs))
	for i := range gs {
		out = append(out, groupView(&gs[i]))
	}
	c.JSON(http.StatusOK, gin.H{"groups": out})
}

// GetGroup: GET /v1/groups/:id
func (h *Handler) GetGroup(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	d, err := h.svc.GetGroup(c.Request.Context(), c.GetInt64("uid"), id)
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"group":        groupView(&d.Group),
		"my_role":      d.MyRole,
		"member_count": d.MemberCount,
	})
}

type inviteRequest struct {
	Email *string `json:"email"`
}

// CreateInvite: POST /v1/groups/:id/invite
func (h *Handler) CreateInvite(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	var req inviteRequest
	_ = c.ShouldBindJSON(&req) // Body ist optional (email)
	raw, inv, err := h.svc.CreateInvite(c.Request.Context(), c.GetInt64("uid"), id, req.Email)
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"invite_token": raw, // an den Eingeladenen weitergeben (Einladungslink)
		"expires_at":   inv.ExpiresAt,
		"group_id":     id,
	})
}

// AcceptInvite: POST /v1/invitations/:token/accept
func (h *Handler) AcceptInvite(c *gin.Context) {
	token := c.Param("token")
	g, err := h.svc.AcceptInvite(c.Request.Context(), c.GetInt64("uid"), token)
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"group": groupView(g)})
}

// parseID liest einen numerischen Pfadparameter; antwortet bei Fehler selbst.
func parseID(c *gin.Context, name string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || id <= 0 {
		fail(c, ErrInvalidInput)
		return 0, false
	}
	return id, true
}
