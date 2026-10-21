package middleware

import (
	"path"
	"strings"
	"time"

	"my-books/backend/config"
	"my-books/backend/logger"

	"github.com/gin-gonic/gin"
)

// isNoisyPath 判断是否高频内容流/静态资源请求：
// 这类请求（封面图、书籍资源、有声书音频、章节文本、前端静态资源）量大且无排查价值，
// 不写入访问日志，避免刷屏与文件膨胀。
func isNoisyPath(p string) bool {
	if strings.HasPrefix(p, "/static/") || strings.HasPrefix(p, "/assets/") {
		return true
	}
	switch strings.ToLower(path.Ext(p)) {
	case ".js", ".css", ".png", ".jpg", ".jpeg", ".gif", ".svg", ".ico",
		".woff", ".woff2", ".ttf", ".mp3", ".m4a", ".pdf":
		return true
	}
	// 章节文本 / 书籍内容资源 / 有声书音频段
	if strings.Contains(p, "/audio/") ||
		strings.HasSuffix(p, "/text") ||
		strings.HasSuffix(p, "/content") ||
		strings.HasSuffix(p, "/resource") {
		return true
	}
	return false
}

// AccessLog 记录 HTTP 访问日志（带网关前缀剥离与噪声过滤）：
//   - 5xx → error 级别；4xx → warn 级别；其余 → info 级别
//   - 通过 config.log_level 控制详细度：默认 info 即可看到所有业务接口调用，
//     debug 时额外输出与 info 相同（slog 统一级别过滤）
func AccessLog() gin.HandlerFunc {
	prefix := config.GatewayPrefix()
	if prefix != "" {
		prefix = "/" + strings.Trim(prefix, "/")
	}
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		reqPath := c.Request.URL.Path
		if prefix != "" && strings.HasPrefix(reqPath, prefix) {
			reqPath = strings.TrimPrefix(reqPath, prefix)
		}
		if isNoisyPath(reqPath) {
			return
		}

		status := c.Writer.Status()
		dur := time.Since(start).Milliseconds()
		fields := []any{
			"method", c.Request.Method,
			"path", reqPath,
			"status", status,
			"dur_ms", dur,
			"ip", c.ClientIP(),
		}
		switch {
		case status >= 500:
			logger.Error("http", fields...)
		case status >= 400:
			logger.Warn("http", fields...)
		default:
			logger.Info("http", fields...)
		}
	}
}
