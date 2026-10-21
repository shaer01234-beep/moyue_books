package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"my-books/backend/config"
	"my-books/backend/middleware"
	"my-books/backend/models"

	"github.com/gin-gonic/gin"
)

// reviewItem 读后感列表项（附书名与封面便于展示）
type reviewItem struct {
	models.BookReview
	BookTitle string `json:"book_title"`
	BookCover string `json:"book_cover"`
}

// reviewForm 读后感创建/更新表单
type reviewForm struct {
	BookID  uint   `json:"book_id" binding:"required"`
	Title   string `json:"title" binding:"required"`
	Content string `json:"content"`
	Rating  int    `json:"rating"`
}

// reviewBind 校验并读取表单
func reviewBind(c *gin.Context) (reviewForm, bool) {
	var form reviewForm
	if err := c.ShouldBindJSON(&form); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误：book_id 与 title 必填"})
		return form, false
	}
	if form.Rating < 0 || form.Rating > 5 {
		form.Rating = 0
	}
	return form, true
}

// reviewBookMap 批量查询书名/封面
func reviewBookMap(userID uint, bookIDs []uint) map[uint]models.Book {
	m := make(map[uint]models.Book)
	if len(bookIDs) == 0 {
		return m
	}
	var books []models.Book
	config.DB.Where("user_id = ? AND id IN ?", userID, bookIDs).Find(&books)
	for _, b := range books {
		m[b.ID] = b
	}
	return m
}

// ListReviews 读后感列表：支持 q 搜索（书名/读后感标题）、book_id 过滤
func ListReviews(c *gin.Context) {
	userID := middleware.CurrentUserID(c)
	q := strings.TrimSpace(c.Query("q"))
	bookID := strings.TrimSpace(c.Query("book_id"))

	db := config.DB.Where("user_id = ?", userID)
	if bookID != "" {
		if bid, err := strconv.ParseUint(bookID, 10, 64); err == nil {
			db = db.Where("book_id = ?", bid)
		}
	}
	if q != "" {
		like := "%" + q + "%"
		db = db.Where("(title LIKE ? OR book_id IN (?))",
			like,
			config.DB.Model(&models.Book{}).Select("id").
				Where("user_id = ? AND title LIKE ?", userID, like))
	}

	var reviews []models.BookReview
	if err := db.Order("updated_at DESC").Find(&reviews).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询读后感失败"})
		return
	}

	bookIDs := make([]uint, 0, len(reviews))
	for _, r := range reviews {
		bookIDs = append(bookIDs, r.BookID)
	}
	bookMap := reviewBookMap(userID, bookIDs)

	items := make([]reviewItem, 0, len(reviews))
	for _, r := range reviews {
		item := reviewItem{BookReview: r}
		if b, ok := bookMap[r.BookID]; ok {
			item.BookTitle = b.Title
			item.BookCover = b.CoverPath
		}
		items = append(items, item)
	}
	c.JSON(http.StatusOK, gin.H{"data": items})
}

// ListBookReviews 某本书的读后感（阅读页使用）
func ListBookReviews(c *gin.Context) {
	userID := middleware.CurrentUserID(c)
	bookID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	var reviews []models.BookReview
	if err := config.DB.Where("user_id = ? AND book_id = ?", userID, bookID).
		Order("updated_at DESC").Find(&reviews).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询读后感失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": reviews})
}

// GetReview 读后感详情
func GetReview(c *gin.Context) {
	userID := middleware.CurrentUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	var review models.BookReview
	if err := config.DB.Where("id = ? AND user_id = ?", id, userID).First(&review).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "读后感不存在"})
		return
	}
	c.JSON(http.StatusOK, review)
}

// CreateReview 创建读后感
func CreateReview(c *gin.Context) {
	userID := middleware.CurrentUserID(c)
	form, ok := reviewBind(c)
	if !ok {
		return
	}
	// 校验图书归属
	var book models.Book
	if err := config.DB.Where("id = ? AND user_id = ?", form.BookID, userID).First(&book).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "图书不存在"})
		return
	}
	review := models.BookReview{
		UserID:  userID,
		BookID:  form.BookID,
		Title:   strings.TrimSpace(form.Title),
		Content: form.Content,
		Rating:  form.Rating,
	}
	if err := config.DB.Create(&review).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建读后感失败"})
		return
	}
	c.JSON(http.StatusOK, review)
}

// UpdateReview 更新读后感
func UpdateReview(c *gin.Context) {
	userID := middleware.CurrentUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	form, ok := reviewBind(c)
	if !ok {
		return
	}
	// 校验图书归属
	var book models.Book
	if err := config.DB.Where("id = ? AND user_id = ?", form.BookID, userID).First(&book).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "图书不存在"})
		return
	}
	var review models.BookReview
	if err := config.DB.Where("id = ? AND user_id = ?", id, userID).First(&review).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "读后感不存在"})
		return
	}
	updates := map[string]interface{}{
		"book_id": form.BookID,
		"title":   strings.TrimSpace(form.Title),
		"content": form.Content,
		"rating":  form.Rating,
	}
	if err := config.DB.Model(&review).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新读后感失败"})
		return
	}
	config.DB.First(&review, id)
	c.JSON(http.StatusOK, review)
}

// DeleteReview 删除读后感
func DeleteReview(c *gin.Context) {
	userID := middleware.CurrentUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	res := config.DB.Where("id = ? AND user_id = ?", id, userID).Delete(&models.BookReview{})
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除读后感失败"})
		return
	}
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "读后感不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
