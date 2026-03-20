package handler

import (
	"errors"
	"net/http"
	"strconv"

	"payment-platform/internal/auth/domain"
	"payment-platform/internal/auth/service"

	"github.com/gin-gonic/gin"
)

const (
	accessTokenCookie  = "access_token"
	refreshTokenCookie = "refresh_token"
	accessTokenMaxAge  = 15 * 60      // 15 minutes
	refreshTokenMaxAge = 7 * 24 * 3600 // 7 days
)

type AuthHandler struct {
	svc *service.AuthService
}

func New(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

func (h *AuthHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/register", h.Register)
	r.POST("/login", h.Login)
	r.POST("/refresh", h.Refresh)
	r.POST("/logout", h.Logout)
	r.GET("/me", h.Me)
	r.POST("/change-password", h.ChangePassword)
}

type RegisterRequest struct {
	Email    string `json:"email"    binding:"required,email"  example:"alice@example.com"`
	Name     string `json:"name"     binding:"required,min=2"  example:"Alice Smith"`
	Password string `json:"password" binding:"required,min=8"  example:"s3cret!Pass"`
}

type LoginRequest struct {
	Email    string `json:"email"    binding:"required,email" example:"alice@example.com"`
	Password string `json:"password" binding:"required"       example:"s3cret!Pass"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" example:"eyJhbGci..."`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" example:"eyJhbGci..."`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"       example:"s3cret!Pass"`
	NewPassword string `json:"new_password" binding:"required,min=8" example:"newS3cret!"`
}

type TokenPairResponse struct {
	AccessToken  string `json:"access_token"  example:"eyJhbGci..."`
	RefreshToken string `json:"refresh_token" example:"eyJhbGci..."`
	ExpiresIn    int64  `json:"expires_in"    example:"900"`
}

type UserResponse struct {
	ID        string `json:"id"         example:"550e8400-e29b-41d4-a716-446655440000"`
	Email     string `json:"email"      example:"alice@example.com"`
	Name      string `json:"name"       example:"Alice Smith"`
	Role      string `json:"role"       example:"customer"`
	CreatedAt string `json:"created_at" example:"2026-03-13T00:00:00Z"`
}

type AuthErrorResponse struct {
	Error string `json:"error" example:"invalid email or password"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var body RegisterRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pair, err := h.svc.Register(c.Request.Context(), body.Email, body.Name, body.Password)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrEmailTaken):
			c.JSON(http.StatusConflict, gin.H{"error": "email already taken"})
		case errors.Is(err, domain.ErrWeakPassword):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "registration failed"})
		}
		return
	}

	setAuthCookies(c, pair)
	c.JSON(http.StatusCreated, toTokenPairResponse(pair))
}

func (h *AuthHandler) Login(c *gin.Context) {
	var body LoginRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pair, err := h.svc.Login(c.Request.Context(), body.Email, body.Password)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "login failed"})
		return
	}

	setAuthCookies(c, pair)
	c.JSON(http.StatusOK, toTokenPairResponse(pair))
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	// Prefer cookie; fall back to JSON body for API clients.
	refreshToken, _ := c.Cookie(refreshTokenCookie)
	if refreshToken == "" {
		var body RefreshRequest
		if err := c.ShouldBindJSON(&body); err != nil || body.RefreshToken == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "refresh token required"})
			return
		}
		refreshToken = body.RefreshToken
	}

	pair, err := h.svc.Refresh(c.Request.Context(), refreshToken)
	if err != nil {
		if errors.Is(err, service.ErrInvalidRefreshToken) {
			clearAuthCookies(c)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired refresh token"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token refresh failed"})
		return
	}

	setAuthCookies(c, pair)
	c.JSON(http.StatusOK, toTokenPairResponse(pair))
}

func (h *AuthHandler) Logout(c *gin.Context) {
	// Prefer cookie; fall back to JSON body for API clients.
	refreshToken, _ := c.Cookie(refreshTokenCookie)
	if refreshToken == "" {
		var body LogoutRequest
		if err := c.ShouldBindJSON(&body); err != nil || body.RefreshToken == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "refresh token required"})
			return
		}
		refreshToken = body.RefreshToken
	}

	if err := h.svc.Logout(c.Request.Context(), refreshToken); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "logout failed"})
		return
	}

	clearAuthCookies(c)
	c.Status(http.StatusNoContent)
}

func (h *AuthHandler) Me(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	user, err := h.svc.GetUser(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch user"})
		return
	}

	c.JSON(http.StatusOK, UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		Role:      user.Role,
		CreatedAt: user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	var body ChangePasswordRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.svc.ChangePassword(c.Request.Context(), userID, body.OldPassword, body.NewPassword); err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidCredentials):
			c.JSON(http.StatusUnauthorized, gin.H{"error": "current password is incorrect"})
		case errors.Is(err, domain.ErrWeakPassword):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "password change failed"})
		}
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *AuthHandler) RegisterAdminRoutes(r *gin.RouterGroup) {
	r.GET("/users", h.ListUsers)
	r.PATCH("/users/:id/role", h.UpdateRole)
}

func (h *AuthHandler) ListUsers(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	users, total, err := h.svc.ListUsers(c.Request.Context(), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list users"})
		return
	}

	type userItem struct {
		ID        string `json:"id"`
		Email     string `json:"email"`
		Name      string `json:"name"`
		Role      string `json:"role"`
		CreatedAt string `json:"created_at"`
	}
	items := make([]userItem, len(users))
	for i, u := range users {
		items[i] = userItem{
			ID:        u.ID,
			Email:     u.Email,
			Name:      u.Name,
			Role:      u.Role,
			CreatedAt: u.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"users":  items,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

type UpdateRoleRequest struct {
	Role string `json:"role" binding:"required" example:"seller"`
}

func (h *AuthHandler) UpdateRole(c *gin.Context) {
	userID := c.Param("id")
	var body UpdateRoleRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.svc.UpdateUserRole(c.Request.Context(), userID, body.Role); err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "role updated"})
}

// setAuthCookies writes httpOnly cookies for the access and refresh tokens.
// Tokens are also included in the JSON body for API clients that cannot use cookies.
func setAuthCookies(c *gin.Context, pair *service.TokenPair) {
	c.SetCookie(accessTokenCookie, pair.AccessToken, accessTokenMaxAge, "/", "", false, true)
	c.SetCookie(refreshTokenCookie, pair.RefreshToken, refreshTokenMaxAge, "/auth/refresh", "", false, true)
}

// clearAuthCookies expires both auth cookies immediately.
func clearAuthCookies(c *gin.Context) {
	c.SetCookie(accessTokenCookie, "", -1, "/", "", false, true)
	c.SetCookie(refreshTokenCookie, "", -1, "/auth/refresh", "", false, true)
}

func toTokenPairResponse(pair *service.TokenPair) TokenPairResponse {
	return TokenPairResponse{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		ExpiresIn:    pair.ExpiresIn,
	}
}
