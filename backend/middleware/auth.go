package middleware

import (
	"net/http"
	"strings"

	"my-books/backend/utils"

	"github.com/gin-gonic/gin"
)

// AuthRequired 校验登录态：校验请求头 Authorization: Bearer <token>，
// 兼容 URL query ?token=：<img>/<iframe> 等原生资源请求无法携带请求头，
// 前端加载封面、EPUB 内嵌图片时通过 query token 完成鉴权（header 优先）。
// 所有入口（统一网关 / 端口直连）均使用应用自身的账号体系，不支持飞牛账号免密登录。
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		token = strings.TrimSpace(token)
		if token == "" {
			token = strings.TrimSpace(c.Query("token"))
		}
		claims, err := utils.ParseToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "未登录或登录已过期"})
			return
		}
		c.Set("userID", claims.UserID)
		c.Set("username", claims.Username)
		c.Next()
	}
}

// CurrentUserID 从上下文中取当前登录用户 ID
func CurrentUserID(c *gin.Context) uint {
	if v, ok := c.Get("userID"); ok {
		if uid, ok := v.(uint); ok {
			return uid
		}
	}
	return 0
}
