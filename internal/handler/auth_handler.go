package handler

import (
	"net/http"
	"strings"

	"github.com/charan/url-shortener/internal/dto"
	"github.com/charan/url-shortener/internal/service"
	"github.com/gin-gonic/gin"
)

const (
	accessCookieName  = "access_token"
	refreshCookieName = "refresh_token"
)

type AuthHandler struct {
	authService         *service.AuthService
	refreshCookieSecure bool
	refreshCookieDomain string
}

func NewAuthHandler(authService *service.AuthService, refreshCookieSecure bool, refreshCookieDomain string) *AuthHandler {
	return &AuthHandler{
		authService:         authService,
		refreshCookieSecure: refreshCookieSecure,
		refreshCookieDomain: strings.TrimSpace(refreshCookieDomain),
	}
}

func (h *AuthHandler) Register(c *gin.Context) {
	c.JSON(http.StatusGone, gin.H{"error": "email registration is disabled; use Sign in with Google"})
}

func (h *AuthHandler) Login(c *gin.Context) {
	c.JSON(http.StatusGone, gin.H{"error": "email login is disabled; use Sign in with Google"})

}

func (h *AuthHandler) Google(c *gin.Context) {
	var req dto.GoogleAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	accessToken, refreshToken, user, err := h.authService.LoginWithGoogle(c.Request.Context(), req.Credential)
	if err != nil {
		if strings.Contains(err.Error(), "not configured") {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "google oauth is not configured"})
			return
		}
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid google credential"})
		return
	}

	h.setRefreshCookie(c, refreshToken)
	h.setAccessCookie(c, accessToken)
	c.JSON(http.StatusOK, gin.H{"user": user})
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	refreshToken, ok := h.readRefreshToken(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing refresh token"})
		return
	}

	accessToken, nextRefreshToken, user, err := h.authService.Refresh(c.Request.Context(), refreshToken)
	if err != nil {
		h.clearAccessCookie(c)
		h.clearRefreshCookie(c)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
		return
	}

	h.setRefreshCookie(c, nextRefreshToken)
	h.setAccessCookie(c, accessToken)
	c.JSON(http.StatusOK, gin.H{"user": user})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	refreshToken, ok := h.readRefreshToken(c)
	if !ok {
		h.clearAccessCookie(c)
		h.clearRefreshCookie(c)
		c.JSON(http.StatusOK, gin.H{"message": "logged out"})
		return
	}

	if err := h.authService.Logout(c.Request.Context(), refreshToken); err != nil {
		h.clearAccessCookie(c)
		h.clearRefreshCookie(c)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
		return
	}

	h.clearAccessCookie(c)
	h.clearRefreshCookie(c)
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

func (h *AuthHandler) Me(c *gin.Context) {
	userID := c.GetString("userID")
	user, err := h.authService.GetUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}

func (h *AuthHandler) setRefreshCookie(c *gin.Context, token string) {
	maxAge := 30 * 24 * 60 * 60
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(refreshCookieName, token, maxAge, "/auth", h.refreshCookieDomain, h.refreshCookieSecure, true)
}

func (h *AuthHandler) setAccessCookie(c *gin.Context, token string) {
	maxAge := 15 * 60
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(accessCookieName, token, maxAge, "/", h.refreshCookieDomain, h.refreshCookieSecure, true)
}

func (h *AuthHandler) clearRefreshCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(refreshCookieName, "", -1, "/auth", h.refreshCookieDomain, h.refreshCookieSecure, true)
}

func (h *AuthHandler) clearAccessCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(accessCookieName, "", -1, "/", h.refreshCookieDomain, h.refreshCookieSecure, true)
}

func (h *AuthHandler) readRefreshToken(c *gin.Context) (string, bool) {
	if cookie, err := c.Cookie(refreshCookieName); err == nil && strings.TrimSpace(cookie) != "" {
		return cookie, true
	}

	var req dto.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err == nil && strings.TrimSpace(req.RefreshToken) != "" {
		return req.RefreshToken, true
	}

	return "", false
}
