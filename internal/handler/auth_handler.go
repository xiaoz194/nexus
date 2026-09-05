package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"nexus/internal/config"
	"nexus/internal/model"
	"nexus/internal/service"
)

// AuthHandler 提供注册、登录、登出与当前用户查询接口。
type AuthHandler struct {
	svc *service.UserService
	cfg config.AuthConfig
}

// NewAuthHandler 构造一个 AuthHandler。
func NewAuthHandler(svc *service.UserService, cfg config.AuthConfig) *AuthHandler {
	return &AuthHandler{svc: svc, cfg: cfg}
}

type registerRequest struct {
	Username    string `json:"username" binding:"required"`
	Password    string `json:"password" binding:"required"`
	DisplayName string `json:"display_name"`
}

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// setSessionCookie 为用户新建会话并写入 HttpOnly cookie。
func (h *AuthHandler) setSessionCookie(c *gin.Context, userID string) error {
	session, err := h.svc.CreateSession(userID)
	if err != nil {
		return err
	}
	maxAge := int(h.cfg.SessionTTLHours) * 3600
	c.SetSameSite(http.SameSiteLaxMode)
	// path=/, domain 空（当前主机）, secure 取配置, httpOnly=true
	c.SetCookie(sessionCookieName, session.Token, maxAge, "/", "", h.cfg.CookieSecure, true)
	return nil
}

// Register 处理 POST /auth/register：注册后直接登录（写会话 cookie）。
func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	user, err := h.svc.Register(req.Username, req.Password, req.DisplayName)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	if err := h.setSessionCookie(c, user.ID); err != nil {
		errorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusCreated, publicUser(user))
}

// Login 处理 POST /auth/login。
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	user, err := h.svc.Authenticate(req.Username, req.Password)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	if err := h.setSessionCookie(c, user.ID); err != nil {
		errorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, publicUser(user))
}

// Logout 处理 POST /auth/logout：删除会话并清 cookie。
func (h *AuthHandler) Logout(c *gin.Context) {
	if token, err := c.Cookie(sessionCookieName); err == nil {
		_ = h.svc.DeleteSession(token)
	}
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(sessionCookieName, "", -1, "/", "", h.cfg.CookieSecure, true)
	c.Status(http.StatusNoContent)
}

// Me 处理 GET /auth/me：返回当前登录用户（走 AuthRequired 之后）。
func (h *AuthHandler) Me(c *gin.Context) {
	user, err := h.svc.LookupSession(mustCookie(c))
	if err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, publicUser(user))
}

// mustCookie 读取会话 cookie（Me 在 AuthRequired 之后调用，cookie 必存在）。
func mustCookie(c *gin.Context) string {
	token, _ := c.Cookie(sessionCookieName)
	return token
}

// publicUser 返回可安全下发前端的用户视图（不含密码哈希）。
func publicUser(u *model.User) gin.H {
	return gin.H{
		"id":           u.ID,
		"username":     u.Username,
		"display_name": u.DisplayName,
		"created_at":   u.CreatedAt,
	}
}
