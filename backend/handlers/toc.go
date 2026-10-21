package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"

	"my-books/backend/config"
	"my-books/backend/middleware"
	"my-books/backend/models"
	"my-books/backend/services"
	"my-books/backend/utils"

	"github.com/gin-gonic/gin"
)

// tocLocks 单飞锁：key=bookID，value=容量 1 的 channel，避免并发重复解析
var tocLocks sync.Map

// pdfEmptyRetried 已对空目录 PDF 重试过解析的 bookID。
// 兼容旧数据：PDF 目录解析功能上线前入库的 PDF 被标记为"已解析"且缓存为空，
// 首次访问时重新解析一次；仍为空（确无书签）则不再重复重试。
var pdfEmptyRetried sync.Map

// GetBookTOC 统一目录接口（EPUB/KEPUB/PDF/MD 等）。
// 已缓存则直接返回目录；未缓存则后台异步解析，立即返回 status=parsing，
// 由前端轮询直到 status=done。
func GetBookTOC(c *gin.Context) {
	id := c.Param("id")
	uid := middleware.CurrentUserID(c)

	var book models.Book
	if err := config.DB.Where("user_id = ?", uid).First(&book, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "book not found"})
		return
	}

	if book.TOCParsed {
		var items []services.TOCItem
		if err := json.Unmarshal([]byte(book.TOC), &items); err != nil || items == nil {
			items = []services.TOCItem{}
		}
		// 兼容旧数据：PDF 目录解析上线前入库的 PDF 缓存为空，
		// 首次访问触发一次重新解析（有书签则补上目录）
		if len(items) == 0 && strings.EqualFold(book.FileType, "pdf") {
			if _, done := pdfEmptyRetried.LoadOrStore(book.ID, true); !done {
				go ParseAndStoreTOC(book)
				c.JSON(http.StatusOK, gin.H{"status": "parsing"})
				return
			}
		}
		c.JSON(http.StatusOK, gin.H{
			"status":   "done",
			"title":    book.Title,
			"chapters": items,
		})
		return
	}

	// 未解析：单飞异步解析（首次请求触发，其余请求直接返回 parsing）
	lockVal, _ := tocLocks.LoadOrStore(book.ID, make(chan struct{}, 1))
	lock := lockVal.(chan struct{})
	select {
	case lock <- struct{}{}:
		go func() {
			defer func() {
				<-lock
				tocLocks.Delete(book.ID)
			}()
			ParseAndStoreTOC(book)
		}()
	default:
		// 已有解析任务在进行
	}

	c.JSON(http.StatusOK, gin.H{"status": "parsing"})
}

// ParseAndStoreTOC 解析书籍目录并写入缓存（上传/扫描入库后异步触发）。
// 注意：该函数会在后台 goroutine 中执行，任何环节 panic 都会终止整个进程，
// 因此必须整体 recover 兜底（PDF 解析库对损坏文件可能 panic）。
func ParseAndStoreTOC(book models.Book) {
	defer func() { _ = recover() }()
	items, _ := services.ExtractTOC(utils.ResolvePath(book.FilePath), book.FileType)
	if items == nil {
		items = []services.TOCItem{}
	}
	b, err := json.Marshal(items)
	if err != nil {
		return
	}
	config.DB.Model(&models.Book{}).Where("id = ?", book.ID).Updates(map[string]interface{}{
		"toc":        string(b),
		"toc_parsed": true,
	})
}
