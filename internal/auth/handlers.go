package auth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler bündelt die HTTP-Endpunkte rund um Auth.
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// errorStatus bildet Service-Fehler auf HTTP-Statuscodes ab.
func errorStatus(err error) int {
	switch {
	case errors.Is(err, ErrInvalidInput):
		return http.StatusBadRequest
	case errors.Is(err, ErrInviteRequired), errors.Is(err, ErrInviteInvalid):
		return http.StatusForbidden
	case errors.Is(err, ErrEmailTaken), errors.Is(err, ErrGroupFull):
		return http.StatusConflict
	case errors.Is(err, ErrInvalidLogin), errors.Is(err, ErrEmailUnverified), errors.Is(err, ErrAccountBlocked):
		return http.StatusUnauthorized
	case errors.Is(err, ErrTokenInvalid):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func fail(c *gin.Context, err error) {
	status := errorStatus(err)
	msg := err.Error()
	if status == http.StatusInternalServerError {
		msg = "interner Fehler" // keine internen Details nach außen
	}
	c.JSON(status, gin.H{"error": msg})
}

type registerRequest struct {
	InviteToken string `json:"invite_token"`
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}

// Register: POST /v1/auth/register
func (h *Handler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, ErrInvalidInput)
		return
	}
	res, err := h.svc.Register(c.Request.Context(), RegisterInput{
		InviteToken: req.InviteToken,
		Email:       req.Email,
		Password:    req.Password,
		DisplayName: req.DisplayName,
	})
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"user": userView(res.User),
		// TODO(Phase 1): in Produktion per E-Mail versenden statt zurückgeben.
		"email_verification_token": res.EmailVerificationToken,
	})
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Login: POST /v1/auth/login
func (h *Handler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, ErrInvalidLogin)
		return
	}
	token, user, err := h.svc.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token, "user": userView(user)})
}

type verifyEmailRequest struct {
	Token string `json:"token"`
}

// VerifyEmail: POST /v1/auth/verify-email
func (h *Handler) VerifyEmail(c *gin.Context) {
	var req verifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, ErrTokenInvalid)
		return
	}
	if err := h.svc.VerifyEmail(c.Request.Context(), req.Token); err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "verified"})
}

// Me: GET /v1/me (geschützt)
func (h *Handler) Me(c *gin.Context) {
	uid := c.GetInt64("uid")
	user, err := h.svc.Me(c.Request.Context(), uid)
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": userView(user)})
}

// userView formt die öffentliche Sicht auf einen Nutzer (ohne Passwort-Hash).
func userView(u *User) gin.H {
	return gin.H{
		"id":             u.ID,
		"email":          u.Email,
		"display_name":   u.DisplayName,
		"role":           u.Role,
		"status":         u.Status,
		"email_verified": u.EmailVerified(),
		"created_at":     u.CreatedAt,
	}
}
