package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"my-books/backend/config"
	"my-books/backend/middleware"
	"my-books/backend/models"
	"my-books/backend/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// generateShareToken 生成随机分享码
func generateShareToken() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// GetShareConfig 公开返回分享所需的服务信息（无需登录）：
// 端口为服务直连端口（飞牛 service_port），前端据此构造「端口直连」的分享链接，不走统一网关
// GET /api/config/public
func GetShareConfig(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"port":           config.Port(),
		"gateway_mode":   config.GatewayMode(),
		"gateway_prefix": config.GatewayPrefix(),
	})
}

// CreateShare 创建图书分享
// POST /api/books/:id/share  body: { "expires_days": 7 }
func CreateShare(c *gin.Context) {
	uid := middleware.CurrentUserID(c)
	bookID := c.Param("id")

	var book models.Book
	if err := config.DB.Where("user_id = ?", uid).First(&book, bookID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "book not found"})
		return
	}

	var input struct {
		ExpiresDays int `json:"expires_days"` // 0 表示永久
	}
	c.ShouldBindJSON(&input)

	var expiresAt *time.Time
	if input.ExpiresDays > 0 {
		t := time.Now().Add(time.Duration(input.ExpiresDays) * 24 * time.Hour)
		expiresAt = &t
	}

	share := models.BookShare{
		UserID:    uid,
		BookID:    book.ID,
		Token:     generateShareToken(),
		ExpiresAt: expiresAt,
	}
	if err := config.DB.Create(&share).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create share"})
		return
	}
	c.JSON(http.StatusCreated, share)
}

// ListShares 查看某本书的分享列表
// GET /api/books/:id/shares
func ListShares(c *gin.Context) {
	uid := middleware.CurrentUserID(c)
	bookID := c.Param("id")

	var book models.Book
	if err := config.DB.Where("user_id = ?", uid).First(&book, bookID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "book not found"})
		return
	}

	var shares []models.BookShare
	config.DB.Where("user_id = ? AND book_id = ?", uid, book.ID).Order("created_at DESC").Find(&shares)
	if shares == nil {
		shares = []models.BookShare{}
	}
	c.JSON(http.StatusOK, gin.H{"data": shares})
}

// RevokeShare 撤销分享（仅分享者可撤销）
// DELETE /api/shares/:token
func RevokeShare(c *gin.Context) {
	uid := middleware.CurrentUserID(c)
	token := c.Param("token")

	var share models.BookShare
	if err := config.DB.Where("token = ? AND user_id = ?", token, uid).First(&share).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "share not found"})
		return
	}
	config.DB.Delete(&share)
	c.JSON(http.StatusOK, gin.H{"message": "revoked"})
}

// GetShareInfo 公开分享信息（无需登录）
// GET /api/shares/:token
func GetShareInfo(c *gin.Context) {
	token := c.Param("token")

	var share models.BookShare
	if err := config.DB.Where("token = ?", token).First(&share).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "share not found"})
		return
	}

	// 过期校验
	if share.ExpiresAt != nil && time.Now().After(*share.ExpiresAt) {
		c.JSON(http.StatusGone, gin.H{"error": "share has expired"})
		return
	}

	var book models.Book
	if err := config.DB.First(&book, share.BookID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "book not found"})
		return
	}

	var sharer models.User
	sharerName := ""
	if err := config.DB.First(&sharer, share.UserID).Error; err == nil {
		sharerName = sharer.Nickname
		if sharerName == "" {
			sharerName = sharer.Username
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"share": gin.H{
			"token":      share.Token,
			"expires_at": share.ExpiresAt,
			"created_at": share.CreatedAt,
		},
		"book": gin.H{
			"id":           book.ID,
			"title":        book.Title,
			"authors":      book.Authors,
			"description":  book.Description,
			"file_type":    book.FileType,
			"file_size":    book.FileSize,
			"pages":        book.Pages,
			"publisher":    book.Publisher,
			"publish_date": book.PublishDate,
			"language":     book.Language,
			"isbn":         book.ISBN,
			"tags":         book.Tags,
			"category":     book.Category,
			"series":       book.Series,
			"has_cover":    book.CoverPath != "",
		},
		"sharer": sharerName,
	})
}

// GetShareCover 公开分享封面（无需登录）
// GET /api/shares/:token/cover
func GetShareCover(c *gin.Context) {
	token := c.Param("token")

	var share models.BookShare
	if err := config.DB.Where("token = ?", token).First(&share).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "share not found"})
		return
	}
	if share.ExpiresAt != nil && time.Now().After(*share.ExpiresAt) {
		c.JSON(http.StatusGone, gin.H{"error": "share has expired"})
		return
	}

	var book models.Book
	if err := config.DB.First(&book, share.BookID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "book not found"})
		return
	}
	if book.CoverPath == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "no cover"})
		return
	}

	data, err := os.ReadFile(utils.ResolvePath(book.CoverPath))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "cover not found"})
		return
	}
	contentType := "image/jpeg"
	if strings.HasSuffix(strings.ToLower(book.CoverPath), ".png") {
		contentType = "image/png"
	}
	c.Data(http.StatusOK, contentType, data)
}

// CheckOwnShare 判断当前登录用户是否是某分享的发起者
// GET /api/shares/:token/own
func CheckOwnShare(c *gin.Context) {
	uid := middleware.CurrentUserID(c)
	token := c.Param("token")

	var share models.BookShare
	if err := config.DB.Where("token = ?", token).First(&share).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "share not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"is_own": share.UserID == uid})
}

// AcceptShare 接收分享：将书复制到自己的书架
// POST /api/shares/:token/accept
func AcceptShare(c *gin.Context) {
	uid := middleware.CurrentUserID(c)
	token := c.Param("token")

	var share models.BookShare
	if err := config.DB.Where("token = ?", token).First(&share).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "share not found"})
		return
	}
	if share.ExpiresAt != nil && time.Now().After(*share.ExpiresAt) {
		c.JSON(http.StatusGone, gin.H{"error": "share has expired"})
		return
	}
	// 不能接收自己分享的书
	if share.UserID == uid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "this is your own book"})
		return
	}

	var book models.Book
	if err := config.DB.First(&book, share.BookID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "book not found"})
		return
	}

	// 去重：检查当前用户是否已拥有相同文件（title + file_size + file_type）
	var existing models.Book
	err := config.DB.Where("user_id = ? AND title = ? AND file_size = ? AND file_type = ?",
		uid, book.Title, book.FileSize, book.FileType).First(&existing).Error
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "already in your library", "book_id": existing.ID})
		return
	}

	// 复制书籍文件
	newFilePath := ""
	if book.FilePath != "" {
		ext := filepath.Ext(book.FilePath)
		uniqueName := uuid.New().String() + ext
		newFilePath = filepath.Join(config.UploadDir(), "books", uniqueName)
		if err := copyFile(utils.ResolvePath(book.FilePath), newFilePath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to copy book file"})
			return
		}
	}

	// 复制封面
	newCoverPath := ""
	if book.CoverPath != "" {
		ext := filepath.Ext(book.CoverPath)
		uniqueName := uuid.New().String() + ext
		newCoverPath = filepath.Join(config.UploadDir(), "covers", uniqueName)
		if err := copyFile(utils.ResolvePath(book.CoverPath), newCoverPath); err != nil {
			os.Remove(newFilePath)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to copy cover"})
			return
		}
	}

	newBook := models.Book{
		UserID:      uid,
		Title:       book.Title,
		Authors:     book.Authors,
		Description: book.Description,
		CoverPath:   newCoverPath,
		FilePath:    newFilePath,
		FileType:    book.FileType,
		FileSize:    book.FileSize,
		Pages:       book.Pages,
		Publisher:   book.Publisher,
		PublishDate: book.PublishDate,
		Language:    book.Language,
		ISBN:        book.ISBN,
		Tags:        book.Tags,
		Category:    book.Category,
		Series:      book.Series,
		SeriesIndex: book.SeriesIndex,
	}
	if err := config.DB.Create(&newBook).Error; err != nil {
		os.Remove(newFilePath)
		os.Remove(newCoverPath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save book"})
		return
	}
	c.JSON(http.StatusCreated, newBook)
}

// copyFile 复制文件
func copyFile(src, dst string) error {
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
