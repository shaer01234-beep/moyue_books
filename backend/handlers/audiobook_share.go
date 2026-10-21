package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"my-books/backend/config"
	"my-books/backend/middleware"
	"my-books/backend/models"
	"my-books/backend/services"
	"my-books/backend/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// generateAudiobookShareToken 生成随机分享码
func generateAudiobookShareToken() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// CreateAudiobookShare 创建有声书分享
// POST /api/audiobooks/:id/share  body: { "expires_days": 7 }
func CreateAudiobookShare(c *gin.Context) {
	uid := middleware.CurrentUserID(c)
	taskID := c.Param("id")

	var task models.AudiobookTask
	if err := config.DB.Where("user_id = ?", uid).First(&task, taskID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "有声书不存在"})
		return
	}
	// 仅已完成的有声书可分享
	if task.Status != "done" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "仅生成完成的有声书可以分享"})
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

	share := models.AudiobookShare{
		UserID:    uid,
		TaskID:    task.ID,
		Token:     generateAudiobookShareToken(),
		ExpiresAt: expiresAt,
	}
	if err := config.DB.Create(&share).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建分享失败"})
		return
	}
	c.JSON(http.StatusCreated, share)
}

// ListAudiobookShares 查看某本有声书的分享列表
// GET /api/audiobooks/:id/shares
func ListAudiobookShares(c *gin.Context) {
	uid := middleware.CurrentUserID(c)
	taskID := c.Param("id")

	var task models.AudiobookTask
	if err := config.DB.Where("user_id = ?", uid).First(&task, taskID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "有声书不存在"})
		return
	}

	var shares []models.AudiobookShare
	config.DB.Where("user_id = ? AND task_id = ?", uid, task.ID).Order("created_at DESC").Find(&shares)
	if shares == nil {
		shares = []models.AudiobookShare{}
	}
	c.JSON(http.StatusOK, gin.H{"data": shares})
}

// RevokeAudiobookShare 撤销有声书分享（仅分享者可撤销）
// DELETE /api/shares/audiobook/:token
func RevokeAudiobookShare(c *gin.Context) {
	uid := middleware.CurrentUserID(c)
	token := c.Param("token")

	var share models.AudiobookShare
	if err := config.DB.Where("token = ? AND user_id = ?", token, uid).First(&share).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "分享不存在"})
		return
	}
	config.DB.Delete(&share)
	c.JSON(http.StatusOK, gin.H{"message": "revoked"})
}

// GetAudiobookShareInfo 公开有声书分享信息（无需登录）
// GET /api/shares/audiobook/:token
func GetAudiobookShareInfo(c *gin.Context) {
	token := c.Param("token")

	var share models.AudiobookShare
	if err := config.DB.Where("token = ?", token).First(&share).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "分享不存在或已失效"})
		return
	}
	if share.ExpiresAt != nil && time.Now().After(*share.ExpiresAt) {
		c.JSON(http.StatusGone, gin.H{"error": "分享已过期"})
		return
	}

	var task models.AudiobookTask
	if err := config.DB.First(&task, share.TaskID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "有声书不存在"})
		return
	}

	var book models.Book
	hasCover := false
	if err := config.DB.First(&book, task.BookID).Error; err == nil {
		hasCover = book.CoverPath != ""
	}

	var sharer models.User
	sharerName := ""
	if err := config.DB.First(&sharer, share.UserID).Error; err == nil {
		sharerName = sharer.Nickname
		if sharerName == "" {
			sharerName = sharer.Username
		}
	}

	uid := middleware.CurrentUserID(c)

	c.JSON(http.StatusOK, gin.H{
		"share": gin.H{
			"token":      share.Token,
			"expires_at": share.ExpiresAt,
			"created_at": share.CreatedAt,
		},
		"audiobook": gin.H{
			"task_id":        task.ID,
			"book_id":        task.BookID,
			"book_title":     task.BookTitle,
			"has_cover":      hasCover,
			"total_segments": task.TotalSegments,
			"voice":          task.Voice,
			"speed_factor":   task.SpeedFactor,
			"engine":         task.Engine,
			"created_at":     task.CreatedAt,
			"finished_at":    task.FinishedAt,
		},
		"sharer": sharerName,
		"is_own": uid > 0 && uid == share.UserID,
	})
}

// GetAudiobookShareCover 公开有声书分享封面（无需登录，封面取自原书）
// GET /api/shares/audiobook/:token/cover
func GetAudiobookShareCover(c *gin.Context) {
	token := c.Param("token")

	var share models.AudiobookShare
	if err := config.DB.Where("token = ?", token).First(&share).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "分享不存在或已失效"})
		return
	}
	if share.ExpiresAt != nil && time.Now().After(*share.ExpiresAt) {
		c.JSON(http.StatusGone, gin.H{"error": "分享已过期"})
		return
	}

	var task models.AudiobookTask
	if err := config.DB.First(&task, share.TaskID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "有声书不存在"})
		return
	}

	var book models.Book
	if err := config.DB.First(&book, task.BookID).Error; err != nil || book.CoverPath == "" {
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

// CheckOwnAudiobookShare 判断当前登录用户是否是某有声书分享的发起者
// GET /api/shares/audiobook/:token/own
func CheckOwnAudiobookShare(c *gin.Context) {
	uid := middleware.CurrentUserID(c)
	token := c.Param("token")

	var share models.AudiobookShare
	if err := config.DB.Where("token = ?", token).First(&share).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "分享不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"is_own": share.UserID == uid})
}

// AcceptAudiobookShare 接收有声书分享：将音频文件复制到当前用户，并创建已完成的任务
// POST /api/shares/audiobook/:token/accept
func AcceptAudiobookShare(c *gin.Context) {
	uid := middleware.CurrentUserID(c)
	token := c.Param("token")

	var share models.AudiobookShare
	if err := config.DB.Where("token = ?", token).First(&share).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "分享不存在或已失效"})
		return
	}
	if share.ExpiresAt != nil && time.Now().After(*share.ExpiresAt) {
		c.JSON(http.StatusGone, gin.H{"error": "分享已过期"})
		return
	}
	// 不能接收自己分享的有声书
	if share.UserID == uid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "这是你分享的有声书"})
		return
	}

	var task models.AudiobookTask
	if err := config.DB.First(&task, share.TaskID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "有声书不存在"})
		return
	}
	// 仅可接收已完成的有声书
	if task.Status != "done" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该有声书尚未生成完成"})
		return
	}

	// 去重：同一本书的完成态有声书已存在则直接返回
	var existing models.AudiobookTask
	err := config.DB.Where("user_id = ? AND book_id = ? AND status = ?",
		uid, task.BookID, "done").First(&existing).Error
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "你的有声书列表中已有这本书", "task_id": existing.ID})
		return
	}

	// 复制封面到接收者（失败不阻断接收，仅封面缺失时列表显示占位图标）
	var book models.Book
	newCoverPath := ""
	if config.DB.First(&book, task.BookID).Error == nil && book.CoverPath != "" {
		ext := filepath.Ext(book.CoverPath)
		uniqueName := uuid.New().String() + ext
		newCoverPath = filepath.Join(config.UploadDir(), "covers", uniqueName)
		if err := copyFile(utils.ResolvePath(book.CoverPath), newCoverPath); err != nil {
			newCoverPath = ""
		}
	}

	// 复制音频目录（音频 + 分段元数据）
	srcDir := services.AudiobookDirFor(share.UserID, task.BookID)
	if _, err := os.Stat(srcDir); err != nil {
		if newCoverPath != "" {
			os.Remove(newCoverPath)
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "分享的音频文件不存在，无法接收"})
		return
	}
	dstDir := services.AudiobookDirFor(uid, task.BookID)
	if err := copyDir(srcDir, dstDir); err != nil {
		if newCoverPath != "" {
			os.Remove(newCoverPath)
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "复制音频文件失败"})
		return
	}

	// 创建新的已完成任务（播放进度从 0 开始）
	newTask := models.AudiobookTask{
		UserID:        uid,
		BookID:        task.BookID,
		BookTitle:     task.BookTitle,
		CoverPath:     newCoverPath,
		Status:        "done",
		Engine:        task.Engine,
		Voice:         task.Voice,
		SpeedFactor:   task.SpeedFactor,
		TextLang:      task.TextLang,
		TotalSegments: task.TotalSegments,
		DoneSegments:  task.TotalSegments,
	}
	if err := config.DB.Create(&newTask).Error; err != nil {
		os.RemoveAll(dstDir)
		if newCoverPath != "" {
			os.Remove(newCoverPath)
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存有声书失败"})
		return
	}
	c.JSON(http.StatusCreated, newTask)
}

// copyDir 递归复制目录
func copyDir(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return copyFile(src, dst)
	}
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if err := copyDir(filepath.Join(src, e.Name()), filepath.Join(dst, e.Name())); err != nil {
			return err
		}
	}
	return nil
}
