package handlers

import (
	"net/http"
	"strings"

	"my-books/backend/config"
	"my-books/backend/middleware"
	"my-books/backend/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CreateCategory 创建分类（支持不选择图书，仅创建空分类；也可同时为图书设置分类；可指定父级分类形成层级）
func CreateCategory(c *gin.Context) {
	var input struct {
		Name    string `json:"name"`
		BookIDs []uint `json:"book_ids"`
		Parent  string `json:"parent"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "分类名称不能为空"})
		return
	}
	uid := middleware.CurrentUserID(c)

	// 校验父级分类存在（若指定）
	if input.Parent != "" {
		var pc models.Category
		if err := config.DB.Where("user_id = ? AND name = ?", uid, input.Parent).First(&pc).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "父级分类不存在"})
			return
		}
	}

	// 记录分类（已存在则忽略），保证空分类可持久化
	var existing models.Category
	err := config.DB.Where("user_id = ? AND name = ?", uid, name).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		cat := models.Category{UserID: uid, Name: name, Parent: input.Parent}
		if cerr := config.DB.Create(&cat).Error; cerr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": cerr.Error()})
			return
		}
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 可选：为选中的图书设置分类
	var affected int64
	if len(input.BookIDs) > 0 {
		res := config.DB.Model(&models.Book{}).
			Where("user_id = ? AND id IN ?", uid, input.BookIDs).
			Update("category", name)
		if res.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": res.Error.Error()})
			return
		}
		affected = res.RowsAffected
	}
	c.JSON(http.StatusOK, gin.H{"message": "created", "affected": affected})
}

// RenameCategory 重命名分类（同步更新该分类下所有图书）
func RenameCategory(c *gin.Context) {
	var input struct {
		OldName string `json:"old_name"`
		NewName string `json:"new_name"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	oldName := strings.TrimSpace(input.OldName)
	newName := strings.TrimSpace(input.NewName)
	if oldName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "old_name 不能为空"})
		return
	}
	if newName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "new_name 不能为空"})
		return
	}
	if oldName == newName {
		c.JSON(http.StatusOK, gin.H{"message": "renamed", "affected": 0})
		return
	}
	uid := middleware.CurrentUserID(c)

	res := config.DB.Model(&models.Book{}).
		Where("user_id = ? AND category = ?", uid, oldName).
		Update("category", newName)
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": res.Error.Error()})
		return
	}

	// 同步分类表记录（含空分类）
	var cat models.Category
	if err := config.DB.Where("user_id = ? AND name = ?", uid, oldName).First(&cat).Error; err == nil {
		cat.Name = newName
		if err := config.DB.Save(&cat).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	// 同步子分类的父级引用
	config.DB.Model(&models.Category{}).
		Where("user_id = ? AND parent = ?", uid, oldName).
		Update("parent", newName)

	c.JSON(http.StatusOK, gin.H{"message": "renamed", "affected": res.RowsAffected})
}

// DeleteCategory 删除分类（支持将图书移动到其他分类）
func DeleteCategory(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name 不能为空"})
		return
	}
	moveTo := strings.TrimSpace(c.Query("move_to"))

	target := ""
	if moveTo != "" && moveTo != name {
		target = moveTo
	}

	res := config.DB.Model(&models.Book{}).
		Where("user_id = ? AND category = ?", middleware.CurrentUserID(c), name).
		Update("category", target)
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": res.Error.Error()})
		return
	}

	// 将子分类的父级指向 moveTo（为空则提升为顶层），避免产生孤儿分类
	config.DB.Model(&models.Category{}).
		Where("user_id = ? AND parent = ?", middleware.CurrentUserID(c), name).
		Update("parent", target)

	// 同步删除分类表记录（空分类）
	config.DB.Where("user_id = ? AND name = ?", middleware.CurrentUserID(c), name).
		Delete(&models.Category{})

	c.JSON(http.StatusOK, gin.H{"message": "deleted", "affected": res.RowsAffected})
}

// AddBooksToCategory 向已有分类追加图书（分类不存在则自动创建，支持空分类）
func AddBooksToCategory(c *gin.Context) {
	name := strings.TrimSpace(c.Param("name"))
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name 不能为空"})
		return
	}
	var input struct {
		BookIDs []uint `json:"book_ids"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(input.BookIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "book_ids 不能为空"})
		return
	}
	uid := middleware.CurrentUserID(c)

	// 确保分类记录存在（向空分类追加图书时也需持久化分类）
	cat := models.Category{UserID: uid, Name: name}
	if err := config.DB.Where("user_id = ? AND name = ?", uid, name).FirstOrCreate(&cat).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	res := config.DB.Model(&models.Book{}).
		Where("user_id = ? AND id IN ?", uid, input.BookIDs).
		Update("category", name)
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": res.Error.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ok", "affected": res.RowsAffected})
}

// RemoveBooksFromCategory 从分类中移除图书（仅当图书当前属于该分类时，将其 category 置空）
func RemoveBooksFromCategory(c *gin.Context) {
	name := strings.TrimSpace(c.Param("name"))
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name 不能为空"})
		return
	}
	var input struct {
		BookIDs []uint `json:"book_ids"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(input.BookIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "book_ids 不能为空"})
		return
	}
	uid := middleware.CurrentUserID(c)

	// 仅清空「当前确实属于该分类」的图书，避免误改其它分类的图书
	res := config.DB.Model(&models.Book{}).
		Where("user_id = ? AND category = ? AND id IN ?", uid, name, input.BookIDs).
		Update("category", "")
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": res.Error.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ok", "affected": res.RowsAffected})
}
