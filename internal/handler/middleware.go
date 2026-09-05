package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"nexus/internal/service"
)

// sessionCookieName 是存放会话 token 的 cookie 名。
const sessionCookieName = "nexus_session"

// contextUserIDKey 是 gin.Context 里存放当前用户 ID 的键。
const contextUserIDKey = "user_id"

// AuthRequired 是鉴权中间件：从 cookie 读会话 token，校验后把
// 当前用户 ID 注入 gin.Context；无效则 401 并中断。
func AuthRequired(userSvc *service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie(sessionCookieName)
		if err != nil {
			errorResponse(c, http.StatusUnauthorized, "未登录")
			c.Abort()
			return
		}
		user, err := userSvc.LookupSession(token)
		if err != nil {
			errorResponse(c, http.StatusUnauthorized, "登录已失效，请重新登录")
			c.Abort()
			return
		}
		c.Set(contextUserIDKey, user.ID)
		c.Next()
	}
}

// currentUserID 从 context 取当前登录用户 ID（AuthRequired 之后必有值）。
func currentUserID(c *gin.Context) string {
	return c.GetString(contextUserIDKey)
}
