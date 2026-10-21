package routes

import (
	"io/fs"
	"mime"
	"net/http"
	"path"
	"path/filepath"
	"strings"
	"time"

	"my-books/backend/config"
	"my-books/backend/handlers"
	"my-books/backend/html"
	"my-books/backend/logger"
	"my-books/backend/middleware"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// SetupRouter 配置路由
func SetupRouter() *gin.Engine {
	// 运行日志：gin 内部日志/panic 恢复输出统一接入 logger（控制台 + <data_dir>/logs/moyue.log）
	gin.SetMode(gin.ReleaseMode)
	gin.DefaultWriter = logger.Writer()
	gin.DefaultErrorWriter = logger.Writer()

	r := gin.New()
	r.Use(middleware.AccessLog())
	r.Use(gin.Recovery())

	// CORS 配置
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// 同时兼容两种入口（同一份二进制、同一份前端构建都能用）：
	//   1) 飞牛统一网关：请求经 /app/moyue 前缀由网关反代到 Unix Socket；
	//   2) 直接 TCP 访问（IP:端口，用于局域网直连 / 书籍分享链接）：请求在根路径。
	// 因此 API 与前端资源都分别在「网关前缀」与「根路径」各注册一套，
	// 无论经网关还是直连，/api 与 /app/moyue/api 都能命中。
	prefixes := gatewayPrefixes()
	for _, p := range prefixes {
		registerAPI(r, p)
	}
	registerFrontend(r, prefixes)

	return r
}

// gatewayPrefixes 返回需要注册路由的前缀集合：始终包含根路径 ""，
// 若配置了统一网关前缀（/app/moyue）则一并加入，去重。
func gatewayPrefixes() []string {
	set := map[string]bool{"": true}
	if gw := config.GatewayPrefix(); gw != "" {
		set["/"+strings.Trim(gw, "/")] = true
	}
	// 固定顺序：根路径在前，网关前缀在后，保证确定性
	out := []string{""}
	for k := range set {
		if k != "" {
			out = append(out, k)
		}
	}
	return out
}

// registerAPI 在指定前缀下注册 API 路由（prefix 为空表示根路径）。
func registerAPI(r *gin.Engine, prefix string) {
	// 静态文件服务（封面和图书文件，位于数据目录下）
	r.Static(prefix+"/static/covers", filepath.Join(config.UploadDir(), "covers"))
	r.Static(prefix+"/static/books", filepath.Join(config.UploadDir(), "books"))

	// API 路由
	api := r.Group(prefix + "/api")
	{
		// 认证（公开）
		auth := api.Group("/auth")
		{
			auth.POST("/register", handlers.Register)
			auth.POST("/login", handlers.Login)
		}

		// 分享服务配置（公开）：返回服务端口，供前端构造端口直连的分享链接
		api.GET("/config/public", handlers.GetShareConfig)

		// 分享（公开预览）
		sharesPublic := api.Group("/shares")
		{
			sharesPublic.GET("/:token", handlers.GetShareInfo)
			sharesPublic.GET("/:token/cover", handlers.GetShareCover)
			// 有声书分享
			sharesPublic.GET("/audiobook/:token", handlers.GetAudiobookShareInfo)
			sharesPublic.GET("/audiobook/:token/cover", handlers.GetAudiobookShareCover)
		}

		// 以下所有接口均需登录
		authed := api.Group("", middleware.AuthRequired())
		{
			authed.GET("/auth/me", handlers.GetMe)
			authed.PUT("/auth/me", handlers.UpdateProfile)
			authed.PUT("/auth/password", handlers.ChangePassword)

			// 图书管理
			books := authed.Group("/books")
			{
				books.GET("", handlers.ListBooks)
				books.GET("/categories", handlers.ListCategories)
				books.POST("/categories", handlers.CreateCategory)
				books.POST("", handlers.UploadBook)
				books.POST("/batch", handlers.UploadBooks)
				books.POST("/scan", handlers.ScanFolder)
				books.GET("/:id/toc", handlers.GetBookTOC)
				books.GET("/:id", handlers.GetBook)
				books.PUT("/:id", handlers.UpdateBook)
				books.DELETE("/:id", handlers.DeleteBook)
				books.GET("/:id/file", handlers.ServeBookFile)
				books.GET("/:id/cover", handlers.GetBookCover)
				books.POST("/:id/cover", handlers.UploadCover)
				books.PATCH("/:id/metadata", handlers.UpdateBookMetadata)
				books.GET("/:id/download", handlers.DownloadBook)
				books.PUT("/:id/progress", handlers.UpdateProgress)
				books.GET("/:id/content", handlers.ServeBookContent)
				books.GET("/:id/resource", handlers.ServeBookResource)
				books.GET("/:id/text", handlers.GetBookChapterText)
				books.GET("/:id/annotations", handlers.ListAnnotations)
				books.POST("/:id/annotations", handlers.CreateAnnotation)
				books.GET("/:id/collections", handlers.GetBookCollections)
				books.PUT("/:id/collections", handlers.SetBookCollections)
				books.POST("/:id/share", handlers.CreateShare)
				books.GET("/:id/shares", handlers.ListShares)
				books.GET("/:id/reviews", handlers.ListBookReviews)
			}

			// 分享
			shares := authed.Group("/shares")
			{
				shares.DELETE("/:token", handlers.RevokeShare)
				shares.GET("/:token/own", handlers.CheckOwnShare)
				shares.POST("/:token/accept", handlers.AcceptShare)
				// 有声书分享
				shares.DELETE("/audiobook/:token", handlers.RevokeAudiobookShare)
				shares.GET("/audiobook/:token/own", handlers.CheckOwnAudiobookShare)
				shares.POST("/audiobook/:token/accept", handlers.AcceptAudiobookShare)
			}

			// 分类管理
			authed.PUT("/categories/rename", handlers.RenameCategory)
			authed.DELETE("/categories/:name", handlers.DeleteCategory)
			authed.POST("/categories/:name/books", handlers.AddBooksToCategory)
		authed.DELETE("/categories/:name/books", handlers.RemoveBooksFromCategory)

			// 批注
			authed.PUT("/annotations/:annotationId", handlers.UpdateAnnotation)
			authed.DELETE("/annotations/:annotationId", handlers.DeleteAnnotation)

			// 阅读历史（会话上报）
			authed.POST("/reading-history", handlers.CreateReadingHistory)

			// 阅读计划
			authed.GET("/plans", handlers.ListPlans)
			authed.POST("/plans", handlers.CreatePlan)
			authed.PUT("/plans/:id", handlers.UpdatePlan)
			authed.DELETE("/plans/:id", handlers.DeletePlan)

			// 读后感（读书笔记）
			reviews := authed.Group("/reviews")
			{
				reviews.GET("", handlers.ListReviews)
				reviews.POST("", handlers.CreateReview)
				reviews.GET("/:id", handlers.GetReview)
				reviews.PUT("/:id", handlers.UpdateReview)
				reviews.DELETE("/:id", handlers.DeleteReview)
			}

			// 统计
			authed.GET("/stats/calendar", handlers.GetCalendarStats)
			authed.GET("/stats/overview", handlers.GetOverviewStats)
			authed.GET("/stats/daily", handlers.GetDailyStats)

			// 收藏夹
			collections := authed.Group("/collections")
			{
				collections.GET("", handlers.ListCollections)
				collections.POST("", handlers.CreateCollection)
				collections.GET("/:id", handlers.GetCollection)
				collections.PUT("/:id", handlers.UpdateCollection)
				collections.DELETE("/:id", handlers.DeleteCollection)
				collections.POST("/:id/books", handlers.AddBooksToCollection)
				collections.DELETE("/:id/books/:bookId", handlers.RemoveBookFromCollection)
			}

			// 媒体库（指定目录扫描入库，不迁移文件，定期巡检记录新增/删除）
			libraries := authed.Group("/libraries")
			{
				libraries.GET("", handlers.ListLibraries)
				libraries.POST("", handlers.CreateLibrary)
				libraries.GET("/:id", handlers.GetLibrary)
				libraries.PUT("/:id", handlers.UpdateLibrary)
				libraries.DELETE("/:id", handlers.DeleteLibrary)
				libraries.POST("/:id/scan", handlers.ScanLibrary)
				libraries.GET("/:id/logs", handlers.ListLibraryScanLogs)
			}

			// 统计（基础）
			authed.GET("/stats", handlers.GetStats)

			// 设置
			authed.GET("/settings", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "settings"})
			})

			// 问题反馈（QQ / 邮件）：163 SMTP 账号与授权码内置于后端源码，
			// 使用者仅填写反馈内容与联系方式，提交后自动发送至开发者邮箱
			feedback := authed.Group("/feedback")
			{
				feedback.GET("/info", handlers.GetFeedbackInfo)
				feedback.POST("/send", handlers.SendFeedbackMail)
			}

			// 语音朗读
			tts := authed.Group("/tts")
			{
				tts.GET("/settings", handlers.GetTTSSettings)
				tts.POST("/synthesize", handlers.SynthesizeTTSHandler)
				tts.GET("/ws", handlers.TTSWebSocketHandler)
			}

			// 有声书
			audiobooks := authed.Group("/audiobooks")
			{
				audiobooks.GET("", handlers.ListAudiobooks)
				audiobooks.POST("", handlers.CreateAudiobook)
				audiobooks.GET("/:id", handlers.GetAudiobook)
				audiobooks.GET("/:id/cover", handlers.GetAudiobookCover)
				audiobooks.POST("/:id/cancel", handlers.CancelAudiobook)
				audiobooks.POST("/:id/pause", handlers.PauseAudiobook)
				audiobooks.POST("/:id/resume", handlers.ResumeAudiobook)
				audiobooks.DELETE("/:id", handlers.DeleteAudiobook)
				audiobooks.PUT("/:id/progress", handlers.UpdateAudiobookProgress)
				audiobooks.GET("/:id/audio/:file", handlers.ServeAudiobookAudio)
				audiobooks.POST("/:id/share", handlers.CreateAudiobookShare)
				audiobooks.GET("/:id/shares", handlers.ListAudiobookShares)
			}
		}
	}
}

// registerFrontend 注册前端静态资源与 SPA 回退。
// 为兼容网关前缀与根路径两种访问方式，NoRoute 会尝试剥离已知前缀后再解析。
func registerFrontend(r *gin.Engine, prefixes []string) {
	dist := html.DistFS()

	r.NoRoute(func(c *gin.Context) {
		reqPath := c.Request.URL.Path

		// 统一网关模式：剥掉网关前缀再按原逻辑处理（/app/moyue -> /）
		for _, p := range prefixes {
			if p != "" && strings.HasPrefix(reqPath, p) {
				reqPath = strings.TrimPrefix(reqPath, p)
				if reqPath == "" {
					reqPath = "/"
				}
				break
			}
		}

		if strings.HasPrefix(reqPath, "/api") || strings.HasPrefix(reqPath, "/static") {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}

		// 去掉开头的 "/"，根路径指向 index.html
		name := strings.TrimPrefix(reqPath, "/")
		if name == "" {
			name = "index.html"
		}

		// 带扩展名的请求视为静态资源，命中失败直接 404
		ext := path.Ext(name)
		if ext != "" && ext != ".html" {
			if data, err := fs.ReadFile(dist, name); err == nil {
				c.Data(http.StatusOK, mime.TypeByExtension(ext), data)
				return
			}
			c.Status(http.StatusNotFound)
			return
		}

		// SPA 回退：返回 index.html
		if index, err := fs.ReadFile(dist, "index.html"); err == nil {
			c.Data(http.StatusOK, "text/html; charset=utf-8", index)
			return
		}
		c.Status(http.StatusServiceUnavailable)
	})
}
