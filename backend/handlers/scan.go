package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"my-books/backend/config"
	"my-books/backend/middleware"
	"my-books/backend/models"
	"my-books/backend/services"
	"my-books/backend/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ScanFolder 扫描服务器本地文件夹，批量导入电子书，通过 SSE 推送实时进度。
// 请求体: {"path": "/abs/path", "recursive": true}
func ScanFolder(c *gin.Context) {
	uid := middleware.CurrentUserID(c)

	var req struct {
		Path     string `json:"path"`
		Category string `json:"category"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Path) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path is required"})
		return
	}

	dir := req.Path
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(utils.ProjectRoot, dir)
	}
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "directory not found: " + dir})
		return
	}

	// ---- 建立 SSE 响应 ----
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

	// ---- 第一遍：仅统计支持的文件数（不保存路径，内存 O(1)），保证进度百分比准确 ----
	uploadsDir := config.UploadDir()
	walkMatched := func(visit func(path string) error) error {
		return filepath.Walk(dir, func(path string, fi os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if fi.IsDir() {
				// 跳过隐藏目录与项目 uploads 目录
				if strings.HasPrefix(fi.Name(), ".") || path == uploadsDir {
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
		send("error", map[string]interface{}{"error": "failed to scan directory"})
		return
	}
	if total == 0 {
		send("done", map[string]interface{}{
			"total": 0, "done": 0, "success": 0, "failed": 0, "exists": 0,
			"message": "该目录下未发现支持的电子书文件",
		})
		return
	}

	send("progress", map[string]interface{}{
		"stage": "scan", "total": total, "done": 0,
		"success": 0, "failed": 0, "exists": 0, "current": "",
	})

	// ---- 并发处理 + 串行入库 ----
	// 设计要点（修复"扫描导入锁库"）：
	//   - worker 池并行做与 DB 无关的繁重工作（哈希 / 拷贝 / 转码 / 元数据 / 封面提取）；
	//   - 单一 DB 写者协程串行执行「查重 + Create」，彻底消除并发写与去重竞态。
	numWorkers := runtime.NumCPU()
	if numWorkers < 1 {
		numWorkers = 1
	}
	if numWorkers > 16 {
		// 上限保护，避免高核心数机器对磁盘造成过大瞬时压力
		numWorkers = 16
	}

	ctx := c.Request.Context()

	// 任务通道、待入库通道与结果通道
	jobs := make(chan string, numWorkers)
	pending := make(chan pendingBook, numWorkers)
	results := make(chan importResult, numWorkers)

	// worker：仅做文件处理（不含任何 DB 操作），产出待入库结构体
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for f := range jobs {
				// 客户端断开则停止处理
				select {
				case <-ctx.Done():
					return
				default:
				}
				pb, err := processBookFile(uid, f, dir, strings.TrimSpace(req.Category))
				if err != nil {
					sendResult(results, ctx, importResult{state: "failed", current: pb.rel})
					continue
				}
				// 投递待入库；若客户端已断开则退出，避免阻塞
				select {
				case pending <- pb:
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	// 所有 worker 退出后关闭 pending，写者 range pending 才能结束并 close(results)，
	// 进而消费者退出、最终 send("done") 才能发出（否则永久阻塞，SSE 无法收尾）。
	go func() {
		wg.Wait()
		close(pending)
	}()

	// 单一 DB 写者：串行查重 + 入库（与 config.DB 单连接池配合，从根本上消除锁冲突）
	go func() {
		defer close(results) // 写者结束即关闭 results，消费者自然退出
		for pb := range pending {
			select {
			case <-ctx.Done():
				return
			default:
			}
			state, _ := createBookRecord(uid, pb)
			sendResult(results, ctx, importResult{state: state, current: pb.rel})
		}
	}()

	// 投递任务：第二遍遍历边发送，通道满时自动阻塞形成背压；客户端断开时安全退出
	go func() {
		defer close(jobs)
		_ = walkMatched(func(path string) error {
			select {
			case jobs <- path:
				return nil
			case <-ctx.Done():
				return filepath.SkipAll
			}
		})
	}()

	// 单一消费者汇总结果并通过 SSE 推送进度（c.Writer 并发写入不安全，统一由此处写入）
	done, success, failed, exists := 0, 0, 0, 0
	for r := range results {
		switch r.state {
		case "imported":
			success++
		case "exists":
			exists++
		default:
			failed++
		}
		done++
		send("progress", map[string]interface{}{
			"stage": "import", "total": total, "done": done,
			"success": success, "failed": failed, "exists": exists,
			"current": r.current,
		})
	}

	wg.Wait()
	send("done", map[string]interface{}{
		"total": total, "done": done,
		"success": success, "failed": failed, "exists": exists,
	})
}

// importResult 单本导入结果（状态 + 当前文件名，用于 SSE 进度推送）
type importResult struct {
	state   string
	current string
}

// sendResult 向结果通道发送，客户端断开时安全退出，避免阻塞写者
func sendResult(ch chan<- importResult, ctx context.Context, r importResult) {
	select {
	case ch <- r:
	case <-ctx.Done():
	}
}

// pendingBook 文件处理完成、待 DB 写者串行入库的中间结构
type pendingBook struct {
	rel       string // 相对扫描根目录的路径，用于进度展示
	hash      string // 文件 sha256，用于去重
	savePath  string // 落盘后的文件绝对路径（重复时需清理）
	coverPath string // 封面绝对路径（重复时需清理）
	book      models.Book
}

// processBookFile 执行与 DB 无关的全部文件处理：哈希、拷贝、转码、元数据、封面提取。
// 返回待入库结构体；任何一步失败返回 error（由 worker 直接记为 failed，不触碰数据库）。
func processBookFile(uid uint, srcPath, baseDir, category string) (pendingBook, error) {
	ext := strings.ToLower(filepath.Ext(srcPath))
	baseName := strings.TrimSuffix(filepath.Base(srcPath), ext)
	isText := ext == ".md" || ext == ".txt"

	// 计算 hash 用于去重（与 UploadBook 相同的查询方式）
	hash := sha256.New()
	f, err := os.Open(srcPath)
	if err != nil {
		return pendingBook{}, err
	}
	io.Copy(hash, f)
	f.Close()
	fileHash := hex.EncodeToString(hash.Sum(nil))

	// 文件名包含完整 hash，保证后续去重可命中
	uniqueName := fileHash + "_" + uuid.New().String() + ext
	savePath := filepath.Join(config.UploadDir(), "books", uniqueName)
	if err := copyLocalFile(srcPath, savePath); err != nil {
		return pendingBook{}, err
	}

	// Markdown / TXT 转为 EPUB，统一走 EPUB 阅读通道
	if ext == ".md" || ext == ".txt" {
		finalPath, finalExt, convErr := maybeConvertToEpub(savePath, baseName)
		if convErr != nil {
			os.Remove(savePath)
			return pendingBook{}, convErr
		}
		savePath = finalPath
		ext = finalExt
	}

	stat, _ := os.Stat(savePath)

	// 提取元数据
	meta, err := services.ExtractMetadata(savePath)
	if err != nil || meta == nil {
		meta = &services.BookMetadata{}
	}

	// 提取封面（EPUB/KEPUB）；文本书无内嵌封面时生成主题色封面
	coverPath := extractBookCover(savePath, ext, meta)
	if coverPath == "" && isText {
		coverPath = generateTextCover(baseName)
	}

	authorsJSON, _ := json.Marshal(meta.Authors)
	tagsJSON, _ := json.Marshal(meta.Tags)

	rel, _ := filepath.Rel(baseDir, srcPath)
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

	return pendingBook{rel: rel, hash: fileHash, savePath: savePath, coverPath: coverPath, book: book}, nil
}

// createBookRecord 在唯一 DB 写者中串行执行「查重 + 入库」，返回状态: imported / exists / failed。
// 查重与插入在同一串行路径内完成，避免并发下的重复插入竞态；重复时清理已拷贝的文件与封面。
func createBookRecord(uid uint, pb pendingBook) (string, uint) {
	var existing models.Book
	if config.DB.Where("user_id = ? AND file_path LIKE ?", uid, "%"+pb.hash+"%").First(&existing).Error == nil {
		os.Remove(pb.savePath)
		if pb.coverPath != "" {
			os.Remove(pb.coverPath)
		}
		return "exists", existing.ID
	}

	if err := config.DB.Create(&pb.book).Error; err != nil {
		os.Remove(pb.savePath)
		if pb.coverPath != "" {
			os.Remove(pb.coverPath)
		}
		return "failed", 0
	}
	// 后台异步解析目录，写入缓存（经单连接池串行，不会引发锁冲突）
	go ParseAndStoreTOC(pb.book)
	return "imported", pb.book.ID
}

func copyLocalFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
