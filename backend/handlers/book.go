package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"my-books/backend/config"
	"my-books/backend/middleware"
	"my-books/backend/models"
	"my-books/backend/utils"

	"github.com/gin-gonic/gin"
)

// ListBooks 获取图书列表
func ListBooks(c *gin.Context) {
	uid := middleware.CurrentUserID(c)
	query := config.DB.Model(&models.Book{}).Where("user_id = ?", uid)

	// 搜索
	if keyword := c.Query("q"); keyword != "" {
		q := "%" + keyword + "%"
		query = query.Where("title LIKE ? OR publisher LIKE ? OR isbn LIKE ? OR authors LIKE ?", q, q, q, q)
	}

	// 按标签筛选
	if tag := c.Query("tag"); tag != "" {
		query = query.Where("tags LIKE ?", "%"+tag+"%")
	}

	// 按分类筛选（支持逗号分隔的多个分类，用于含子分类的递归查询）
	if category := c.Query("category"); category != "" {
		parts := strings.Split(category, ",")
		clean := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				clean = append(clean, p)
			}
		}
		if len(clean) > 0 {
			query = query.Where("category IN ?", clean)
		}
	}

	// 按媒体库筛选（为空表示不过滤）
	if lid := c.Query("library_id"); lid != "" {
		query = query.Where("library_id = ?", lid)
	}

	// 按阅读状态筛选（配合书架统计块点击）
	switch c.Query("status") {
	case "reading":
		query = query.Where("progress > 0 AND progress < 100")
	case "finished":
		query = query.Where("progress >= 100")
	case "unread":
		query = query.Where("(progress = 0 OR progress IS NULL) AND last_read IS NULL")
	}

	// 排序（白名单校验，防止任意列名注入）
	allowedSorts := map[string]string{
		"created_at":   "created_at",
		"updated_at":   "updated_at",
		"title":        "title",
		"authors":      "authors",
		"publisher":    "publisher",
		"category":     "category",
		"last_read":    "last_read",
		"series_index": "series_index",
	}
	sort, ok := allowedSorts[c.DefaultQuery("sort", "created_at")]
	if !ok {
		sort = "created_at"
	}
	order := "desc"
	if c.DefaultQuery("order", "desc") == "asc" {
		order = "asc"
	}
	query = query.Order(sort + " " + order)

	// 分页
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	var total int64
	query.Count(&total)

	var results []models.Book
	query.Offset((page - 1) * pageSize).Limit(pageSize).Find(&results)

	c.JSON(http.StatusOK, gin.H{
		"data":      results,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetBook 获取单本图书详情
func GetBook(c *gin.Context) {
	id := c.Param("id")
	uid := middleware.CurrentUserID(c)
	var book models.Book
	if err := config.DB.Where("user_id = ?", uid).First(&book, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "book not found"})
		return
	}
	c.JSON(http.StatusOK, book)
}

// UpdateBook 更新图书信息
func UpdateBook(c *gin.Context) {
	id := c.Param("id")
	uid := middleware.CurrentUserID(c)
	var book models.Book
	if err := config.DB.Where("user_id = ?", uid).First(&book, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "book not found"})
		return
	}

	var input struct {
		Title       string   `json:"title"`
		Authors     []string `json:"authors"`
		Description string   `json:"description"`
		Publisher   string   `json:"publisher"`
		PublishDate string   `json:"publish_date"`
		Language    string   `json:"language"`
		ISBN        string   `json:"isbn"`
		Tags        []string `json:"tags"`
		Category    *string  `json:"category"`
		Series      string   `json:"series"`
		SeriesIndex float64  `json:"series_index"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if input.Title != "" {
		updates["title"] = input.Title
	}
	if len(input.Authors) > 0 {
		authorsJSON, _ := json.Marshal(input.Authors)
		updates["authors"] = string(authorsJSON)
	}
	if input.Description != "" {
		updates["description"] = input.Description
	}
	if input.Publisher != "" {
		updates["publisher"] = input.Publisher
	}
	if input.PublishDate != "" {
		updates["publish_date"] = input.PublishDate
	}
	if input.Language != "" {
		updates["language"] = input.Language
	}
	if input.ISBN != "" {
		updates["isbn"] = input.ISBN
	}
	if len(input.Tags) > 0 {
		tagsJSON, _ := json.Marshal(input.Tags)
		updates["tags"] = string(tagsJSON)
	}
	if input.Category != nil {
		updates["category"] = *input.Category
	}
	if input.Series != "" {
		updates["series"] = input.Series
	}
	updates["series_index"] = input.SeriesIndex

	config.DB.Model(&book).Updates(updates)
	c.JSON(http.StatusOK, book)
}

// ListCategories 获取图书分类及数量（合并自建分类表与图书聚合，空分类也会返回）
func ListCategories(c *gin.Context) {
	uid := middleware.CurrentUserID(c)

	// 1. 图书聚合分类及数量
	var agg []struct {
		Category string `json:"-"`
		Count    int64  `json:"count"`
	}
	config.DB.Model(&models.Book{}).
		Select("category, COUNT(*) as count").
		Where("user_id = ? AND category <> ''", uid).
		Group("category").
		Scan(&agg)
	sort.Slice(agg, func(i, j int) bool { return agg[i].Count > agg[j].Count })

	// 2. 自建分类表（含空分类，按创建时间）
	var cats []models.Category
	config.DB.Where("user_id = ?", uid).Order("created_at ASC").Find(&cats)

	countMap := make(map[string]int64, len(agg))
	for _, a := range agg {
		countMap[a.Category] = a.Count
	}

	type CategoryItem struct {
		Category string `json:"category"`
		Count    int64  `json:"count"`
		Parent   string `json:"parent"`
	}
	seen := make(map[string]bool, len(cats)+len(agg))
	results := make([]CategoryItem, 0, len(cats)+len(agg))
	for _, cat := range cats {
		results = append(results, CategoryItem{Category: cat.Name, Count: countMap[cat.Name], Parent: cat.Parent})
		seen[cat.Name] = true
	}
	for _, a := range agg {
		if seen[a.Category] {
			continue
		}
		results = append(results, CategoryItem{Category: a.Category, Count: a.Count, Parent: ""})
	}
	c.JSON(http.StatusOK, gin.H{"data": results})
}

// DeleteBook 删除图书
func DeleteBook(c *gin.Context) {
	id := c.Param("id")
	uid := middleware.CurrentUserID(c)
	var book models.Book
	if err := config.DB.Where("user_id = ?", uid).First(&book, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "book not found"})
		return
	}

	// 删除文件
	// 媒体库图书的 FilePath 指向用户原始文件，绝不删除源文件；仅清理本地封面缩略图。
	if book.LibraryID == nil && book.FilePath != "" {
		os.Remove(utils.ResolvePath(book.FilePath))
	}
	if book.CoverPath != "" {
		os.Remove(utils.ResolvePath(book.CoverPath))
	}

	// 清理该书的所有分享记录
	config.DB.Where("book_id = ?", book.ID).Delete(&models.BookShare{})

	config.DB.Delete(&book)
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// UpdateProgress 更新阅读进度
func UpdateProgress(c *gin.Context) {
	id := c.Param("id")
	uid := middleware.CurrentUserID(c)
	var book models.Book
	if err := config.DB.Where("user_id = ?", uid).First(&book, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "book not found"})
		return
	}

	var input struct {
		Progress    float64 `json:"progress"`
		CurrentPage int     `json:"current_page"`
		TTSLocation string  `json:"tts_location"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	book.Progress = input.Progress
	book.CurrentPage = input.CurrentPage
	// 朗读断点：仅在传值非空时更新（老前端不带此字段，避免误清空历史断点）
	if input.TTSLocation != "" {
		book.TTSLocation = input.TTSLocation
	}
	now := time.Now()
	book.LastRead = &now
	// 列级更新：阅读中每 ~800ms 保存一次进度，若用 Save 全行保存，
	// 会把含 TOC 目录缓存（可达数十 KB JSON）的整行重写，单连接 SQLite 下
	// 大行写会排队阻塞其他查询（打开书籍/批注/目录）。只写变化的列。
	updates := map[string]interface{}{
		"progress":     input.Progress,
		"current_page": input.CurrentPage,
		"last_read":    now,
	}
	if input.TTSLocation != "" {
		updates["tts_location"] = input.TTSLocation
	}
	config.DB.Model(&book).Updates(updates)
	c.JSON(http.StatusOK, book)
}

// GetBookCover 获取图书封面
func GetBookCover(c *gin.Context) {
	id := c.Param("id")
	uid := middleware.CurrentUserID(c)
	var book models.Book
	if err := config.DB.Where("user_id = ?", uid).First(&book, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "book not found"})
		return
	}

	if book.CoverPath == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "no cover"})
		return
	}

	// 检测文件类型
	data, err := os.ReadFile(utils.ResolvePath(book.CoverPath))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "cover not found"})
		return
	}

	contentType := "image/jpeg"
	lower := strings.ToLower(book.CoverPath)
	switch {
	case strings.HasSuffix(lower, ".png"):
		contentType = "image/png"
	case strings.HasSuffix(lower, ".svg"):
		contentType = "image/svg+xml"
	}
	c.Data(http.StatusOK, contentType, data)
}

// DownloadBook 下载图书文件
func DownloadBook(c *gin.Context) {
	id := c.Param("id")
	uid := middleware.CurrentUserID(c)
	var book models.Book
	if err := config.DB.Where("user_id = ?", uid).First(&book, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "book not found"})
		return
	}

	if book.FilePath == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}

	c.FileAttachment(utils.ResolvePath(book.FilePath), filepath.Base(book.FilePath))
}

// ServeBookFile 按图书 ID 提供原始文件（内联展示，用于 PDF 阅读器等），
// 直接读取 book.FilePath（媒体库图书即指向用户原始文件，无需迁移）。
func ServeBookFile(c *gin.Context) {
	id := c.Param("id")
	uid := middleware.CurrentUserID(c)
	var book models.Book
	if err := config.DB.Where("user_id = ?", uid).First(&book, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "book not found"})
		return
	}
	if book.SourceMissing {
		c.JSON(http.StatusGone, gin.H{"error": "源文件已不存在，可能已被移动或删除"})
		return
	}
	if book.FilePath == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}
	fp := utils.ResolvePath(book.FilePath)
	info, err := os.Stat(fp)
	if err != nil {
		c.JSON(http.StatusGone, gin.H{"error": "源文件已不存在"})
		return
	}
	// HTTP 缓存协商：ETag（修改时间 + 大小）+ 强制重新验证。
	// 重复打开同一本书时浏览器带 If-None-Match 命中 304，不再重复传输整本书文件；
	// 源文件被替换/移动后 modtime 变化，ETag 随之改变，不会读到旧内容。
	// c.File 内部走 http.ServeFile → ServeContent，会自动利用上面的 ETag
	// 处理条件请求（304）与 Range 断点/分段请求。
	c.Header("ETag", fmt.Sprintf(`"%x-%x"`, info.ModTime().UnixNano(), info.Size()))
	c.Header("Cache-Control", "private, no-cache")
	c.File(fp)
}

// GetStats 获取统计信息
func GetStats(c *gin.Context) {
	uid := middleware.CurrentUserID(c)
	var totalBooks int64
	var totalPages int64
	var totalSize int64
	var readingCount int64
	var finishedCount int64

	config.DB.Model(&models.Book{}).Where("user_id = ?", uid).Count(&totalBooks)
	config.DB.Model(&models.Book{}).Where("user_id = ?", uid).Select("COALESCE(SUM(pages), 0)").Scan(&totalPages)
	config.DB.Model(&models.Book{}).Where("user_id = ?", uid).Select("COALESCE(SUM(file_size), 0)").Scan(&totalSize)
	config.DB.Model(&models.Book{}).Where("user_id = ? AND progress > 0 AND progress < 100", uid).Count(&readingCount)
	config.DB.Model(&models.Book{}).Where("user_id = ? AND progress >= 100", uid).Count(&finishedCount)

	c.JSON(http.StatusOK, gin.H{
		"total_books":    totalBooks,
		"total_pages":    totalPages,
		"total_size":     totalSize,
		"reading_count":  readingCount,
		"finished_count": finishedCount,
	})
}
