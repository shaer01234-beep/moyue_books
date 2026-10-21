package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"my-books/backend/config"
	"my-books/backend/middleware"
	"my-books/backend/models"
	"my-books/backend/services"
	"my-books/backend/utils"

	"github.com/gin-gonic/gin"
)

// libraryScanning 防止同一媒体库并发扫描（手动 + 监听触发共享）。
var libraryScanning sync.Map

// ===================== 媒体库 CRUD =====================

// ListLibraries 列出当前用户全部媒体库（附带实时图书数）。
func ListLibraries(c *gin.Context) {
	uid := middleware.CurrentUserID(c)
	var libs []models.Library
	config.DB.Where("user_id = ?", uid).Order("created_at DESC").Find(&libs)
	for i := range libs {
		var cnt int64
		config.DB.Model(&models.Book{}).Where("user_id = ? AND library_id = ?", uid, libs[i].ID).Count(&cnt)
		libs[i].BookCount = int(cnt)
	}
	c.JSON(http.StatusOK, gin.H{"data": libs})
}

// CreateLibrary 新建媒体库。path 必须是服务器上已存在的目录。
func CreateLibrary(c *gin.Context) {
	uid := middleware.CurrentUserID(c)
	var req struct {
		Name      string `json:"name"`
		Path      string `json:"path"`
		Recursive *bool  `json:"recursive"`
		AutoScan  *bool  `json:"auto_scan"`
		Category  string `json:"category"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请填写媒体库名称"})
		return
	}
	dir := req.Path
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(utils.ProjectRoot, dir)
	}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "目录不存在或不可访问: " + dir})
		return
	}

	now := time.Now()
	lib := models.Library{
		UserID:    uid,
		Name:      strings.TrimSpace(req.Name),
		Path:      dir,
		Recursive: req.Recursive == nil || *req.Recursive,
		AutoScan:  req.AutoScan == nil || *req.AutoScan,
		Category:  strings.TrimSpace(req.Category),
		// 初始 LastScanAt 设为创建时间；启用监听后，文件变动才会触发首次自动扫描，
		// 不会与用户的"立即扫描"抢锁（返回 409）。
		LastScanAt: &now,
	}
	if err := config.DB.Create(&lib).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败"})
		return
	}
	// 启用自动扫描时立即开始监听该目录
	WatchLibrary(&lib)
	c.JSON(http.StatusCreated, lib)
}

// GetLibrary 媒体库详情（附带实时图书数）。
func GetLibrary(c *gin.Context) {
	uid := middleware.CurrentUserID(c)
	var lib models.Library
	if err := config.DB.Where("user_id = ?", uid).First(&lib, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "媒体库不存在"})
		return
	}
	var cnt int64
	config.DB.Model(&models.Book{}).Where("user_id = ? AND library_id = ?", uid, lib.ID).Count(&cnt)
	lib.BookCount = int(cnt)
	c.JSON(http.StatusOK, lib)
}

// UpdateLibrary 更新媒体库配置。
func UpdateLibrary(c *gin.Context) {
	uid := middleware.CurrentUserID(c)
	var lib models.Library
	if err := config.DB.Where("user_id = ?", uid).First(&lib, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "媒体库不存在"})
		return
	}
	var req struct {
		Name      *string `json:"name"`
		Path      *string `json:"path"`
		Recursive *bool   `json:"recursive"`
		AutoScan  *bool   `json:"auto_scan"`
		Category  *string `json:"category"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	updates := map[string]interface{}{}
	if req.Name != nil && strings.TrimSpace(*req.Name) != "" {
		updates["name"] = strings.TrimSpace(*req.Name)
	}
	if req.Path != nil && strings.TrimSpace(*req.Path) != "" {
		dir := *req.Path
		if !filepath.IsAbs(dir) {
			dir = filepath.Join(utils.ProjectRoot, dir)
		}
		if info, err := os.Stat(dir); err != nil || !info.IsDir() {
			c.JSON(http.StatusBadRequest, gin.H{"error": "目录不存在或不可访问: " + dir})
			return
		}
		updates["path"] = dir
	}
	if req.Recursive != nil {
		updates["recursive"] = *req.Recursive
	}
	if req.AutoScan != nil {
		updates["auto_scan"] = *req.AutoScan
	}
	if req.Category != nil {
		newCat := strings.TrimSpace(*req.Category)
		if newCat != lib.Category {
			// 分类变更：将该媒体库下所有图书同步到新分类（确保「书籍放到对应分类下」）
			config.DB.Model(&models.Book{}).
				Where("user_id = ? AND library_id = ?", uid, lib.ID).
				Update("category", newCat)
		}
		updates["category"] = newCat
	}
	if len(updates) > 0 {
		config.DB.Model(&lib).Updates(updates)
	}
	config.DB.Where("id = ?", lib.ID).First(&lib)
	// 配置（路径 / 递归 / 自动扫描）可能变更，重建该库的目录监听
	var reloaded models.Library
	if config.DB.Where("id = ?", lib.ID).First(&reloaded).Error == nil {
		UnwatchLibrary(reloaded.ID)
		WatchLibrary(&reloaded)
	}
	c.JSON(http.StatusOK, lib)
}

// DeleteLibrary 删除媒体库：仅删除数据库记录（媒体库图书、扫描日志），不触碰源文件。
func DeleteLibrary(c *gin.Context) {
	uid := middleware.CurrentUserID(c)
	var lib models.Library
	if err := config.DB.Where("user_id = ?", uid).First(&lib, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "媒体库不存在"})
		return
	}
	config.DB.Where("library_id = ?", lib.ID).Delete(&models.LibraryScanLog{})
	// 仅删除数据库中的图书记录，绝不删除用户原始文件
	config.DB.Where("user_id = ? AND library_id = ?", uid, lib.ID).Delete(&models.Book{})
	// 停止该媒体库的目录监听
	UnwatchLibrary(lib.ID)
	config.DB.Delete(&lib)
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// ListLibraryScanLogs 媒体库扫描记录（分页）。
func ListLibraryScanLogs(c *gin.Context) {
	uid := middleware.CurrentUserID(c)
	id := c.Param("id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	var total int64
	config.DB.Model(&models.LibraryScanLog{}).Where("user_id = ? AND library_id = ?", uid, id).Count(&total)
	var logs []models.LibraryScanLog
	config.DB.Where("user_id = ? AND library_id = ?", uid, id).
		Order("scan_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&logs)
	c.JSON(http.StatusOK, gin.H{"data": logs, "total": total, "page": page, "page_size": pageSize})
}

// ===================== 扫描（手动 SSE / 定时） =====================

// ScanLibrary 手动触发扫描：通过 SSE 实时推送进度。
func ScanLibrary(c *gin.Context) {
	uid := middleware.CurrentUserID(c)
	var lib models.Library
	if err := config.DB.Where("user_id = ?", uid).First(&lib, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "媒体库不存在"})
		return
	}
	if _, loaded := libraryScanning.LoadOrStore(lib.ID, true); loaded {
		c.JSON(http.StatusConflict, gin.H{"error": "该媒体库正在扫描中"})
		return
	}
	defer libraryScanning.Delete(lib.ID)

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)
	c.Writer.Flush()

	send := func(event string, data map[string]interface{}) {
		b, _ := json.Marshal(data)
		fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, b)
		c.Writer.Flush()
	}

	config.DB.Model(&lib).Update("status", "scanning")
	logEntry, err := runLibraryScan(&lib, send)
	if err != nil {
		send("error", map[string]interface{}{"error": err.Error()})
		send("done", map[string]interface{}{
			"total": 0, "added": 0, "updated": 0, "deleted": 0, "failed": 0, "error": err.Error(),
		})
		return
	}
	send("done", map[string]interface{}{
		"total":   logEntry.TotalFound,
		"added":   logEntry.Added,
		"updated": logEntry.Updated,
		"deleted": logEntry.Deleted,
		"failed":  0,
	})
}

// ===================== 扫描核心逻辑 =====================

type libJob struct{ path string }

type libPending struct {
	path    string
	rel     string
	modTime int64
	book    models.Book
}

type libResult struct {
	state   string // added / updated / exists / recovered / failed
	path    string
	current string
}

// existingBook 媒体库扫描时加载的已有图书快照，用于去重 / 变更 / 删除检测。
type existingBook struct {
	ID       uint   `gorm:"column:id"`
	FilePath string `gorm:"column:file_path"`
	ModTime  int64  `gorm:"column:source_mod_time"`
	Missing  bool   `gorm:"column:source_missing"`
	// found 在扫描期间由串行写者就地标记，避免额外维护一份全量 foundSet。
	found bool
}

// scanDetailMax 扫描详情（新增 / 更新 / 删除列表）最多保留的条目数，
// 防止上万文件时详情 JSON 撑爆内存与数据库字段。
const scanDetailMax = 200

type libraryScanSummary struct {
	Added        int
	Updated      int
	Deleted      int
	Failed       int
	AddedList    []string
	UpdatedList  []string
	DeletedList  []string
}

// runLibraryScan 执行一次媒体库扫描：
//   - 扫描目录收集支持的电子书文件；
//   - 新文件：提取元数据 / 封面后入库（FilePath 指向源文件，不迁移）；
//   - 已存在且变更：更新元数据；
//   - 源文件消失：标记 source_missing 并记录删除；
//   - 写入一条扫描记录，并更新媒体库元数据。
//
// emit 为 nil 时（定时扫描）不推送 SSE。
func runLibraryScan(lib *models.Library, emit func(event string, data map[string]interface{})) (*models.LibraryScanLog, error) {
	uid := lib.UserID
	baseDir := lib.Path
	if info, err := os.Stat(baseDir); err != nil || !info.IsDir() {
		return nil, fmt.Errorf("目录不存在或不可访问: %s", baseDir)
	}

	summary := &libraryScanSummary{}

	// ---- 1. 加载已有图书（去重 / 变更 / 删除检测） ----
	// 仅保留「路径 -> 已有记录」映射，扫描期间在串行写者中就地标记 found，
	// 避免在内存中额外维护全量切片与 foundSet 三套结构。
	existingByPath := make(map[string]*existingBook)
	{
		var raw []existingBook
		if err := config.DB.Model(&models.Book{}).
			Where("user_id = ? AND library_id = ?", uid, lib.ID).
			Select("id, file_path, source_mod_time, source_missing").
			Find(&raw).Error; err != nil {
			return nil, err
		}
		for i := range raw {
			e := raw[i]
			existingByPath[e.FilePath] = &e
		}
		raw = nil // 释放临时切片，路径字符串由映射复用，不重复占用
	}

	// ---- 2. 第一遍：仅统计支持的文件数（不保存路径，内存 O(1)），保证进度百分比准确 ----
	walkMatched := func(visit func(path string) error) error {
		return filepath.Walk(baseDir, func(path string, fi os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if fi.IsDir() {
				if !lib.Recursive && path != baseDir {
					return filepath.SkipDir
				}
				if strings.HasPrefix(fi.Name(), ".") {
					return filepath.SkipDir
				}
				return nil
			}
			if !SupportedFormats[strings.ToLower(filepath.Ext(path))] {
				return nil
			}
			return visit(path)
		})
	}
	total := 0
	if err := walkMatched(func(string) error { total++; return nil }); err != nil {
		return nil, err
	}

	if emit != nil {
		emit("progress", map[string]interface{}{
			"stage": "scan", "total": total, "done": 0,
			"added": 0, "updated": 0, "deleted": 0, "failed": 0, "current": "",
		})
	}

	// ---- 3. 流式遍历 + 并发解析 + 串行入库 ----
	// 第二遍遍历边投递 jobs，通道仅按并发数缓冲，内存占用与文件总数无关。
	ctx := context.Background()
	numWorkers := runtime.NumCPU()
	if numWorkers < 1 {
		numWorkers = 1
	}
	if numWorkers > 16 {
		numWorkers = 16
	}

	jobs := make(chan libJob, numWorkers)
	pending := make(chan libPending, numWorkers)
	results := make(chan libResult, numWorkers)

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				pb, err := buildLibraryPending(uid, j.path, baseDir, lib.ID)
				if err != nil {
					select {
					case results <- libResult{state: "failed", path: j.path, current: filepath.Base(j.path)}:
					case <-ctx.Done():
					}
					continue
				}
				select {
				case pending <- pb:
				case <-ctx.Done():
				}
			}
		}()
	}

	// 所有 worker 退出后关闭 pending，让串行写者的 range pending 结束，
	// 进而 close(results)、主循环才能退出并发出 done 事件（否则会永久阻塞，SSE 无法收尾）。
	go func() {
		wg.Wait()
		close(pending)
	}()

	go func() {
		defer close(results)
		for pb := range pending {
			res := importLibraryBook(uid, pb, lib.Category, existingByPath, summary)
			select {
			case results <- res:
			case <-ctx.Done():
			}
		}
	}()

	// 生产者：第二遍遍历边投递，通道满时自动阻塞形成背压，内存有界。
	go func() {
		defer close(jobs)
		_ = walkMatched(func(path string) error {
			select {
			case jobs <- libJob{path: path}:
				return nil
			case <-ctx.Done():
				return filepath.SkipAll
			}
		})
	}()

	done := 0
	for r := range results {
		done++
		if emit != nil {
			emit("progress", map[string]interface{}{
				"stage": "import", "total": total, "done": done,
				"added": summary.Added, "updated": summary.Updated,
				"deleted": summary.Deleted, "failed": summary.Failed,
				"current": r.current,
			})
		}
	}
	wg.Wait()

	// ---- 3. 删除检测：源文件消失（仅遍历既有映射，不再维护额外集合） ----
	for _, ex := range existingByPath {
		if ex.found || ex.Missing {
			continue // 已找到或此前已标记缺失，跳过
		}
		config.DB.Model(&models.Book{}).Where("id = ?", ex.ID).Update("source_missing", true)
		var b models.Book
		config.DB.Select("title").First(&b, ex.ID)
		summary.Deleted++
		if len(summary.DeletedList) < scanDetailMax {
			summary.DeletedList = append(summary.DeletedList, b.Title)
		}
	}

	// ---- 4. 写扫描记录 + 更新媒体库元数据 ----
	var bc int64
	config.DB.Model(&models.Book{}).Where("user_id = ? AND library_id = ?", uid, lib.ID).Count(&bc)

	detail, _ := json.Marshal(map[string]interface{}{
		"added":   summary.AddedList,
		"updated": summary.UpdatedList,
		"deleted": summary.DeletedList,
	})
	logEntry := models.LibraryScanLog{
		UserID:     uid,
		LibraryID:  lib.ID,
		ScanAt:     time.Now(),
		TotalFound: total,
		Added:      summary.Added,
		Updated:    summary.Updated,
		Deleted:    summary.Deleted,
		Detail:     string(detail),
	}
	config.DB.Create(&logEntry)

	now := time.Now()
	resJSON, _ := json.Marshal(map[string]interface{}{
		"total": total, "added": summary.Added, "updated": summary.Updated,
		"deleted": summary.Deleted, "failed": summary.Failed,
	})
	config.DB.Model(lib).Updates(map[string]interface{}{
		"last_scan_at": &now,
		"last_result":  string(resJSON),
		"book_count":   int(bc),
		"status":       "idle",
	})

	return &logEntry, nil
}

// buildLibraryPending 解析单个源文件（不复制文件，仅提取元数据 / 封面缩略图）。
func buildLibraryPending(uid uint, srcPath, baseDir string, libID uint) (libPending, error) {
	ext := strings.ToLower(filepath.Ext(srcPath))
	baseName := strings.TrimSuffix(filepath.Base(srcPath), ext)
	fi, err := os.Stat(srcPath)
	if err != nil {
		return libPending{}, err
	}
	meta, mErr := services.ExtractMetadata(srcPath)
	if mErr != nil || meta == nil {
		meta = &services.BookMetadata{Title: baseName}
	}
	if meta.Title == "" {
		meta.Title = baseName
	}
	isText := ext == ".md" || ext == ".txt"
	// 封面为派生缩略图（写入 covers 目录），不属于"迁移源文件"
	coverPath := extractBookCover(srcPath, ext, meta)
	if coverPath == "" && isText {
		coverPath = generateTextCover(baseName)
	}
	authorsJSON, _ := json.Marshal(meta.Authors)
	tagsJSON, _ := json.Marshal(meta.Tags)
	rel, _ := filepath.Rel(baseDir, srcPath)

	book := models.Book{
		UserID:         uid,
		Title:          meta.Title,
		Authors:        string(authorsJSON),
		Description:    meta.Description,
		Publisher:      meta.Publisher,
		PublishDate:    meta.PublishDate,
		Language:       meta.Language,
		ISBN:           meta.ISBN,
		Tags:           string(tagsJSON),
		Series:         meta.Series,
		SeriesIndex:    meta.SeriesIndex,
		FilePath:       srcPath, // 关键：直接指向源文件，不迁移
		CoverPath:      coverPath,
		FileType:       strings.TrimPrefix(ext, "."),
		FileSize:       fi.Size(),
		Pages:          meta.Pages,
		LibraryID:      &libID,
		LibraryRelPath: rel,
		SourceMissing:  false,
		SourceModTime:  fi.ModTime().Unix(),
	}
	return libPending{path: srcPath, rel: rel, modTime: fi.ModTime().Unix(), book: book}, nil
}

// importLibraryBook 在串行写者中执行去重 / 入库 / 更新。
// libCategory 为媒体库绑定的分类（可能为空），非空时入库图书归属该分类。
func importLibraryBook(uid uint, pb libPending, libCategory string, existingByPath map[string]*existingBook, sum *libraryScanSummary) libResult {
	ex, found := existingByPath[pb.path]
	if !found {
		// 新文件入库：归属媒体库绑定的分类（如有）
		pb.book.Category = libCategory
		if err := config.DB.Create(&pb.book).Error; err != nil {
			sum.Failed++
			return libResult{state: "failed", path: pb.path, current: pb.book.Title}
		}
		go ParseAndStoreTOC(pb.book)
		sum.Added++
		if len(sum.AddedList) < scanDetailMax {
			sum.AddedList = append(sum.AddedList, pb.book.Title)
		}
		return libResult{state: "added", path: pb.path, current: pb.book.Title}
	}

	// 已存在：标记 found，供删除检测阶段识别
	ex.found = true

	// 已存在：源文件此前被标记缺失，现在恢复
	if ex.Missing {
		config.DB.Model(&models.Book{}).Where("id = ?", ex.ID).
			Updates(map[string]interface{}{"source_missing": false, "source_mod_time": pb.modTime})
		return libResult{state: "recovered", path: pb.path, current: pb.book.Title}
	}

	// 内容（修改时间）变更：更新元数据
	if ex.ModTime != pb.modTime {
		updates := map[string]interface{}{
			"title":           pb.book.Title,
			"authors":         pb.book.Authors,
			"description":     pb.book.Description,
			"publisher":       pb.book.Publisher,
			"publish_date":    pb.book.PublishDate,
			"language":        pb.book.Language,
			"isbn":            pb.book.ISBN,
			"tags":            pb.book.Tags,
			"series":          pb.book.Series,
			"series_index":    pb.book.SeriesIndex,
			"file_size":       pb.book.FileSize,
			"pages":           pb.book.Pages,
			"file_type":       pb.book.FileType,
			"source_mod_time": pb.modTime,
		}
		// 媒体库绑定了分类时，同步更新图书分类（确保「书籍放到对应分类下」）
		if libCategory != "" {
			updates["category"] = libCategory
		}
		var old models.Book
		config.DB.Select("cover_path").First(&old, ex.ID)
		if old.CoverPath == "" && pb.book.CoverPath != "" {
			updates["cover_path"] = pb.book.CoverPath
		} else if pb.book.CoverPath != "" {
			// 已有封面，清理本次新提取的孤儿缩略图
			os.Remove(pb.book.CoverPath)
		}
		config.DB.Model(&models.Book{}).Where("id = ?", ex.ID).Updates(updates)
		sum.Updated++
		if len(sum.UpdatedList) < scanDetailMax {
			sum.UpdatedList = append(sum.UpdatedList, pb.book.Title)
		}
		return libResult{state: "updated", path: pb.path, current: pb.book.Title}
	}

	return libResult{state: "exists", path: pb.path, current: pb.book.Title}
}
