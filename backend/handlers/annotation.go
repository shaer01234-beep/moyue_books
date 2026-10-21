package handlers

import (
	"net/http"
	"strconv"

	"my-books/backend/config"
	"my-books/backend/middleware"
	"my-books/backend/models"

	"github.com/gin-gonic/gin"
)

// ListAnnotations 获取图书的批注/划线列表
func ListAnnotations(c *gin.Context) {
	bookID := c.Param("id")
	uid := middleware.CurrentUserID(c)
	query := config.DB.Model(&models.Annotation{}).Where("book_id = ? AND user_id = ?", bookID, uid)

	// 按类型筛选
	if typ := c.Query("type"); typ != "" {
		query = query.Where("type = ?", typ)
	}

	var annotations []models.Annotation
	query.Order("created_at desc").Find(&annotations)
	c.JSON(http.StatusOK, annotations)
}

// CreateAnnotation 创建批注/划线
func CreateAnnotation(c *gin.Context) {
	bookID := c.Param("id")
	var input struct {
		Type     string `json:"type" binding:"required"`
		Content  string `json:"content"`
		Note     string `json:"note"`
		Position string `json:"position"`
		Color    string `json:"color"`
		Style    string `json:"style"`
		Page     int    `json:"page"`
		Chapter  string `json:"chapter"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	bid, _ := strconv.ParseUint(bookID, 10, 32)
	annotation := models.Annotation{
		UserID:   middleware.CurrentUserID(c),
		BookID:   uint(bid),
		Type:     input.Type,
		Content:  input.Content,
		Note:     input.Note,
		Position: input.Position,
		Color:    input.Color,
		Style:    input.Style,
		Page:     input.Page,
		Chapter:  input.Chapter,
	}

	if err := config.DB.Create(&annotation).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create annotation"})
		return
	}
	c.JSON(http.StatusCreated, annotation)
}

// UpdateAnnotation 更新批注
func UpdateAnnotation(c *gin.Context) {
	id := c.Param("annotationId")
	uid := middleware.CurrentUserID(c)
	var annotation models.Annotation
	if err := config.DB.Where("user_id = ?", uid).First(&annotation, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "annotation not found"})
		return
	}

	var input struct {
		Content  string `json:"content"`
		Note     string `json:"note"`
		Color    string `json:"color"`
		Style    string `json:"style"`
		Position string `json:"position"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if input.Content != "" {
		updates["content"] = input.Content
	}
	updates["note"] = input.Note
	if input.Color != "" {
		updates["color"] = input.Color
	}
	if input.Style != "" {
		updates["style"] = input.Style
	}
	if input.Position != "" {
		updates["position"] = input.Position
	}

	config.DB.Model(&annotation).Updates(updates)
	c.JSON(http.StatusOK, annotation)
}

// DeleteAnnotation 删除批注
func DeleteAnnotation(c *gin.Context) {
	id := c.Param("annotationId")
	uid := middleware.CurrentUserID(c)
	var annotation models.Annotation
	if err := config.DB.Where("user_id = ?", uid).First(&annotation, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "annotation not found"})
		return
	}
	config.DB.Delete(&annotation)
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
