package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"my-books/backend/config"
	"my-books/backend/middleware"
	"my-books/backend/models"
	"my-books/backend/services"
	"my-books/backend/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// SupportedFormats 支持的文件格式（仅保留可在线阅读的格式）
// .md / .txt 会在上传/扫描时自动转换为 EPUB，因此也在此接受
// .mobi 作为 MOBI/KF8 容器，复用 AZW3 解析器阅读
var SupportedFormats = map[string]bool{
	".epub": true, ".kepub": true, ".pdf": true, ".azw3": true, ".mobi": true,
	".md": true, ".txt": true,
}

// maybeConvertToEpub 若源文件为 Markdown/TXT，则转换为 EPUB 并返回新的路径与扩展名；
// 否则原样返回。转换成功后删除原始文本文件（EPUB 已包含全部内容）。
func maybeConvertToEpub(savePath, title string) (string, string, error) {
	ext := strings.ToLower(filepath.Ext(savePath))
	if ext != ".md" && ext != ".txt" {
		return savePath, ext, nil
	}
	epubPath, err := services.ConvertTextToEPUB(savePath, title, strings.TrimPrefix(ext, "."), config.UploadDir())
	if err != nil {
		return "", "", err
	}
	if err := os.Remove(savePath); err != nil {
		log.Printf("[upload] remove original text file failed: %v", err)
	}
	return epubPath, ".epub", nil
}

// extractBookCover 提取书籍封面写入 covers 目录，返回封面绝对路径；无封面返回空串。
// 支持格式：.epub/.kepub（ZIP 容器，KEPUB 为 EPUB 变种，解析一致）、.azw3（MOBI/KF8 容器），
// 优先用元数据阶段的 CoverData，缺失时再打开文件二次尝试。
func extractBookCover(savePath, ext string, meta *services.BookMetadata) string {
	writeCover := func(data []byte) string {
		if len(data) == 0 {
			return ""
		}
		coverPath := filepath.Join(config.UploadDir(), "covers", uuid.New().String()+".jpg")
		os.WriteFile(coverPath, data, 0644)
		return coverPath
	}
	if meta != nil && len(meta.CoverData) > 0 {
		return writeCover(meta.CoverData)
	}
	if ext == ".epub" || ext == ".kepub" {
		epub, err := services.OpenEPUB(savePath)
		if err != nil {
			return ""
		}
		defer epub.Close()
		coverData, err := epub.GetCover()
		if err != nil {
			return ""
		}
		return writeCover(coverData)
	}
	if ext == ".azw3" || ext == ".mobi" {
		azw3, err := services.OpenAZW3(savePath)
		if err != nil {
			return ""
		}
		defer azw3.Close()
		coverData, err := azw3.GetCover()
		if err != nil {
			return ""
		}
		return writeCover(coverData)
	}
	return ""
}

// generateTextCover 为没有内嵌封面的文本书生成一张主题色 SVG 封面。
// 主题色由书名哈希决定，保证同一本书封面稳定。返回封面绝对路径，失败返回空串。
func generateTextCover(title string) string {
	if title == "" {
		title = "文本书"
	}
	hue := titleHue(title)
	c1 := hslToHex(hue, 62, 46)
	c2 := hslToHex((hue+38)%360, 68, 34)

	var tb strings.Builder
	lines := wrapTitle(title, 9)
	y := 160 - (len(lines)-1)*30
	for _, ln := range lines {
		tb.WriteString(fmt.Sprintf(
			`<text x="28" y="%d" font-size="26" font-weight="700" fill="#ffffff" font-family="PingFang SC, Microsoft YaHei, sans-serif">%s</text>`,
			y, html.EscapeString(ln)))
		y += 38
	}

	svg := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" width="300" height="420" viewBox="0 0 300 420">
  <defs>
    <linearGradient id="g" x1="0" y1="0" x2="1" y2="1">
      <stop offset="0" stop-color="%s"/>
      <stop offset="1" stop-color="%s"/>
    </linearGradient>
  </defs>
  <rect width="300" height="420" fill="url(#g)"/>
  <rect x="16" y="16" width="268" height="388" fill="none" stroke="rgba(255,255,255,0.35)" stroke-width="2" rx="10"/>
  %s
  <text x="28" y="384" font-size="14" fill="rgba(255,255,255,0.85)" font-family="PingFang SC, Microsoft YaHei, sans-serif">文本书</text>
</svg>`, c1, c2, tb.String())

	coverPath := filepath.Join(config.UploadDir(), "covers", uuid.New().String()+".svg")
	if err := os.MkdirAll(filepath.Dir(coverPath), 0o755); err != nil {
		return ""
	}
	if err := os.WriteFile(coverPath, []byte(svg), 0o644); err != nil {
		return ""
	}
	return coverPath
}

// titleHue 由书名稳定地推导一个色相（0-359）
func titleHue(s string) int {
	h := sha256.Sum256([]byte(s))
	sum := (int(h[0]) + int(h[1])*7 + int(h[2])*13) % 360
	if sum < 0 {
		sum += 360
	}
	return sum
}

// hslToHex 将 HSL 颜色转为 #RRGGBB
func hslToHex(h, s, l int) string {
	hs, ls, ll := float64(h)/360.0, float64(s)/100.0, float64(l)/100.0
	var r, g, b float64
	if ls == 0 {
		r, g, b = ll, ll, ll
	} else {
		hue2rgb := func(p, q, t float64) float64 {
			if t < 0 {
				t += 1
			}
			if t > 1 {
				t -= 1
			}
			if t < 1.0/6 {
				return p + (q-p)*6*t
			}
			if t < 1.0/2 {
				return q
			}
			if t < 2.0/3 {
				return p + (q-p)*(2.0/3-t)*6
			}
			return p
		}
		var q float64
		if ll < 0.5 {
			q = ll * (1 + ls)
		} else {
			q = ll + ls - ll*ls
		}
		p := 2*ll - q
		r = hue2rgb(p, q, hs+1.0/3)
		g = hue2rgb(p, q, hs)
		b = hue2rgb(p, q, hs-1.0/3)
	}
	return fmt.Sprintf("#%02X%02X%02X", int(r*255+0.5), int(g*255+0.5), int(b*255+0.5))
}

// wrapTitle 按字符数将书名折行，最多 4 行
func wrapTitle(title string, max int) []string {
	runes := []rune(title)
	if len(runes) <= max {
		return []string{title}
	}
	var res []string
	for i := 0; i < len(runes); i += max {
		end := i + max
		if end > len(runes) {
			end = len(runes)
		}
		res = append(res, string(runes[i:end]))
	}
	if len(res) > 4 {
		res = res[:4]
		res[3] = res[3] + "…"
	}
	return res
}

// UploadBook 上传单本图书
func UploadBook(c *gin.Context) {
	uid := middleware.CurrentUserID(c)
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no file uploaded"})
		return
	}
	defer file.Close()
	category := strings.TrimSpace(c.PostForm("category"))

	// 检查格式
	ext := strings.ToLower(filepath.Ext(header.Filename))
	baseName := strings.TrimSuffix(header.Filename, ext)
	isText := ext == ".md" || ext == ".txt"
	if !SupportedFormats[ext] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported file format: " + ext})
		return
	}

	// 计算文件 hash 用于去重
	hash := sha256.New()
	file.Seek(0, io.SeekStart)
	io.Copy(hash, file)
	fileHash := hex.EncodeToString(hash.Sum(nil))
	file.Seek(0, io.SeekStart)

	// 检查重复（通过 hash，仅限当前用户）
	var existing models.Book
	if config.DB.Where("user_id = ? AND file_path LIKE ?", uid, "%"+fileHash+"%").First(&existing).Error == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "file already exists", "book_id": existing.ID})
		return
	}

	// 保存文件
	uniqueName := uuid.New().String() + ext
	savePath := filepath.Join(config.UploadDir(), "books", uniqueName)
	// 上传落盘日志：确认文件实际写入位置（排查 TRIM_APPSHARE 是否生效）
	log.Printf("[upload] saving book to: %s (upload dir source: %s)", savePath, config.UploadDir())
	out, err := os.Create(savePath)
	if err != nil {
		log.Printf("[upload] create file failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save file"})
		return
	}
	defer out.Close()
	file.Seek(0, io.SeekStart)
	io.Copy(out, file)

	// Markdown / TXT 转为 EPUB，统一走 EPUB 阅读通道
	if ext == ".md" || ext == ".txt" {
		finalPath, finalExt, convErr := maybeConvertToEpub(savePath, strings.TrimSuffix(header.Filename, ext))
		if convErr != nil {
			os.Remove(savePath)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to convert text to epub: " + convErr.Error()})
			return
		}
		savePath = finalPath
		ext = finalExt
	}

	// 获取文件信息
	stat, _ := os.Stat(savePath)

	// 提取元数据
	meta, err := services.ExtractMetadata(savePath)
	if err != nil || meta == nil {
		meta = &services.BookMetadata{Title: baseName}
	}

	// 提取封面（EPUB）；文本书无内嵌封面时生成主题色封面
	coverPath := extractBookCover(savePath, ext, meta)
	if coverPath == "" && isText {
		coverPath = generateTextCover(baseName)
	}

	// 构建图书记录
	authorsJSON, _ := json.Marshal(meta.Authors)
	tagsJSON, _ := json.Marshal(meta.Tags)

	book := models.Book{
		UserID:      uid,
		Title:       baseName,
		Authors:     string(authorsJSON),
		Description: meta.Description,
		Publisher:   meta.Publisher,
		PublishDate: meta.PublishDate,
		Language:    meta.Language,
		ISBN:        meta.ISBN,
		Tags:        string(tagsJSON),
		Series:      meta.Series,
		SeriesIndex: meta.SeriesIndex,
		Category:    category,
		FilePath:    savePath,
		CoverPath:   coverPath,
		FileType:    strings.TrimPrefix(ext, "."),
		FileSize:    stat.Size(),
		Pages:       meta.Pages,
	}

	if err := config.DB.Create(&book).Error; err != nil {
		// 清理文件
		os.Remove(savePath)
		if coverPath != "" {
			os.Remove(coverPath)
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save to database"})
		return
	}

	// 后台异步解析目录，写入缓存
	go ParseAndStoreTOC(book)

	c.JSON(http.StatusCreated, book)
}

// UploadBooks 批量上传图书
func UploadBooks(c *gin.Context) {
	uid := middleware.CurrentUserID(c)
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	files := form.File["files"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no files uploaded"})
		return
	}
	category := strings.TrimSpace(c.PostForm("category"))

	var results []map[string]interface{}
	for _, fileHeader := range files {
		file, _ := fileHeader.Open()
		func() {
			defer file.Close()

			ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
			baseName := strings.TrimSuffix(fileHeader.Filename, ext)
			isText := ext == ".md" || ext == ".txt"
			if !SupportedFormats[ext] {
				results = append(results, map[string]interface{}{
					"filename": fileHeader.Filename,
					"success":  false,
					"error":    "unsupported format",
				})
				return
			}

			uniqueName := uuid.New().String() + ext
			savePath := filepath.Join(config.UploadDir(), "books", uniqueName)
			out, err := os.Create(savePath)
			if err != nil {
				results = append(results, map[string]interface{}{
					"filename": fileHeader.Filename,
					"success":  false,
					"error":    "save failed",
				})
				return
			}
			defer out.Close()
			io.Copy(out, file)

			// Markdown / TXT 转为 EPUB
			if ext == ".md" || ext == ".txt" {
				finalPath, finalExt, convErr := maybeConvertToEpub(savePath, baseName)
				if convErr != nil {
					os.Remove(savePath)
					results = append(results, map[string]interface{}{
						"filename": fileHeader.Filename,
						"success":  false,
						"error":    "convert failed",
					})
					return
				}
				savePath = finalPath
				ext = finalExt
			}

			meta, _ := services.ExtractMetadata(savePath)
			if meta == nil {
				meta = &services.BookMetadata{Title: baseName}
			}

			stat, _ := os.Stat(savePath)

			// 提取封面（EPUB）；文本书无内嵌封面时生成主题色封面
			coverPath := extractBookCover(savePath, ext, meta)
			if coverPath == "" && isText {
				coverPath = generateTextCover(baseName)
			}

			authorsJSON, _ := json.Marshal(meta.Authors)
			tagsJSON, _ := json.Marshal(meta.Tags)

			book := models.Book{
				UserID:      uid,
				Title:       baseName,
				Authors:     string(authorsJSON),
				Description: meta.Description,
				Publisher:   meta.Publisher,
				PublishDate: meta.PublishDate,
				Language:    meta.Language,
				ISBN:        meta.ISBN,
				Tags:        string(tagsJSON),
				Series:      meta.Series,
				SeriesIndex: meta.SeriesIndex,
				Category:    category,
				FilePath:    savePath,
				CoverPath:   coverPath,
				FileType:    strings.TrimPrefix(ext, "."),
				FileSize:    stat.Size(),
				Pages:       meta.Pages,
			}

			if err := config.DB.Create(&book).Error; err != nil {
				os.Remove(savePath)
				if coverPath != "" {
					os.Remove(coverPath)
				}
				results = append(results, map[string]interface{}{
					"filename": fileHeader.Filename,
					"success":  false,
					"error":    "database failed",
				})
				return
			}

			// 后台异步解析目录，写入缓存
			go ParseAndStoreTOC(book)

			results = append(results, map[string]interface{}{
				"filename": fileHeader.Filename,
				"success":  true,
				"book_id":  book.ID,
			})
		}()
	}

	c.JSON(http.StatusOK, gin.H{"results": results})
}

// UploadCover 上传封面
func UploadCover(c *gin.Context) {
	id := c.Param("id")
	uid := middleware.CurrentUserID(c)
	var book models.Book
	if err := config.DB.Where("user_id = ?", uid).First(&book, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "book not found"})
		return
	}

	file, header, err := c.Request.FormFile("cover")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no cover uploaded"})
		return
	}
	defer file.Close()

	// 只接受图片
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only jpg/png allowed"})
		return
	}

	uniqueName := uuid.New().String() + ext
	// 使用基于项目根目录的绝对路径，避免数据库存相对路径导致其他进程/工作目录读取失败
	savePath := filepath.Join(config.UploadDir(), "covers", uniqueName)
	out, err := os.Create(savePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save cover"})
		return
	}
	defer out.Close()
	io.Copy(out, file)

	// 删除旧封面
	if book.CoverPath != "" {
		os.Remove(utils.ResolvePath(book.CoverPath))
	}

	book.CoverPath = savePath
	config.DB.Save(&book)
	c.JSON(http.StatusOK, book)
}

// UpdateBookMetadata 手动触发重新提取元数据
func UpdateBookMetadata(c *gin.Context) {
	id := c.Param("id")
	uid := middleware.CurrentUserID(c)
	var book models.Book
	if err := config.DB.Where("user_id = ?", uid).First(&book, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "book not found"})
		return
	}

	meta, err := services.ExtractMetadata(utils.ResolvePath(book.FilePath))
	if err != nil || meta == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to extract metadata"})
		return
	}

	updates := map[string]interface{}{}
	if meta.Title != "" {
		updates["title"] = meta.Title
	}
	if len(meta.Authors) > 0 {
		authorsJSON, _ := json.Marshal(meta.Authors)
		updates["authors"] = string(authorsJSON)
	}
	if meta.Description != "" {
		updates["description"] = meta.Description
	}
	if meta.Publisher != "" {
		updates["publisher"] = meta.Publisher
	}
	if meta.PublishDate != "" {
		updates["publish_date"] = meta.PublishDate
	}
	if meta.Language != "" {
		updates["language"] = meta.Language
	}
	if meta.ISBN != "" {
		updates["isbn"] = meta.ISBN
	}
	if len(meta.Tags) > 0 {
		tagsJSON, _ := json.Marshal(meta.Tags)
		updates["tags"] = string(tagsJSON)
	}
	if meta.Series != "" {
		updates["series"] = meta.Series
		updates["series_index"] = meta.SeriesIndex
	}

	config.DB.Model(&book).Updates(updates)
	c.JSON(http.StatusOK, book)
}

// bookReader 阅读器内容接口（EPUB 与 AZW3 共用）
type bookReader interface {
	GetContent(int) ([]byte, error)
	GetResource(string) ([]byte, error)
	GetChapterList() []map[string]interface{}
	Close()
}

// openBookReader 按格式打开阅读器
func openBookReader(filePath, fileType string) (bookReader, error) {
	switch fileType {
	case "epub", "kepub":
		return services.OpenEPUB(filePath)
	case "azw3", "mobi":
		return services.OpenAZW3(filePath)
	}
	return nil, fmt.Errorf("unsupported file type: %s", fileType)
}

// ServeBookContent 提供 EPUB/AZW3 内容文件（用于阅读器）
func ServeBookContent(c *gin.Context) {
	id := c.Param("id")
	uid := middleware.CurrentUserID(c)
	var book models.Book
	if err := config.DB.Where("user_id = ?", uid).First(&book, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "book not found"})
		return
	}

	if book.FileType != "epub" && book.FileType != "kepub" && book.FileType != "azw3" && book.FileType != "mobi" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only epub/azw3/mobi content serving is supported"})
		return
	}

	chapterStr := c.Query("chapter")
	if chapterStr == "" {
		// 返回章节列表
		switch book.FileType {
		case "azw3", "mobi":
			azw3, err := services.OpenAZW3(utils.ResolvePath(book.FilePath))
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open azw3"})
				return
			}
			defer azw3.Close()
			c.JSON(http.StatusOK, gin.H{
				"title":    azw3.Title,
				"authors":  azw3.Authors,
				"chapters": azw3.GetChapterList(),
			})
			return
		default:
			epub, err := services.OpenEPUB(utils.ResolvePath(book.FilePath))
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open epub"})
				return
			}
			defer epub.Close()
			c.JSON(http.StatusOK, gin.H{
				"title":    epub.Title,
				"authors":  epub.Authors,
				"chapters": epub.GetChapterList(),
			})
			return
		}
	}

	// 返回指定章节内容
	chapter, _ := strconv.Atoi(chapterStr)
	reader, err := openBookReader(utils.ResolvePath(book.FilePath), book.FileType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open book"})
		return
	}
	defer reader.Close()

	content, err := reader.GetContent(chapter)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Type", "application/xhtml+xml; charset=utf-8")
	c.String(http.StatusOK, string(content))
}

// ServeBookResource 提供 EPUB/AZW3 内嵌资源（图片等），供 <img> 标签直接加载。
// 鉴权通过 URL query ?token= 完成（<img> 无法携带 Authorization 请求头）。
func ServeBookResource(c *gin.Context) {
	id := c.Param("id")
	uid := middleware.CurrentUserID(c)
	var book models.Book
	if err := config.DB.Where("user_id = ?", uid).First(&book, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "book not found"})
		return
	}
	if book.FileType != "epub" && book.FileType != "kepub" && book.FileType != "azw3" && book.FileType != "mobi" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only epub/azw3/mobi resource serving is supported"})
		return
	}
	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path is required"})
		return
	}
	reader, err := openBookReader(utils.ResolvePath(book.FilePath), book.FileType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open book"})
		return
	}
	defer reader.Close()
	data, err := reader.GetResource(path)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.Data(http.StatusOK, http.DetectContentType(data), data)
}
