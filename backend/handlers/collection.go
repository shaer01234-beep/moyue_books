package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"my-books/backend/config"
	"my-books/backend/middleware"
	"my-books/backend/models"

	"github.com/gin-gonic/gin"
)

// ListCollections 获取收藏夹列表（含图书数量与封面缩略图）
func ListCollections(c *gin.Context) {
	uid := middleware.CurrentUserID(c)
	var collections []models.Collection
	config.DB.Where("user_id = ?", uid).Order("sort_order ASC, id DESC").Find(&collections)

	// 统计每夹图书数 + 最近加入的图书（用于封面墙）
	type Row struct {
		CollectionID uint
		BookID       uint
		CreatedAt    time.Time
	}
	var rows []Row
	config.DB.Table("collection_books").
		Select("collection_id, book_id, created_at").
		Where("user_id = ?", uid).
		Order("created_at DESC").
		Scan(&rows)

	counts := map[uint]int64{}
	covers := map[uint][]uint{}
	for _, r := range rows {
		counts[r.CollectionID]++
		if len(covers[r.CollectionID]) < 4 {
			covers[r.CollectionID] = append(covers[r.CollectionID], r.BookID)
		}
	}

	type CollectionResp struct {
		models.Collection
		BookCount int64  `json:"book_count"`
		Covers    []uint `json:"covers"`
	}
	result := make([]CollectionResp, 0, len(collections))
	for _, col := range collections {
		result = append(result, CollectionResp{
			Collection: col,
			BookCount:  counts[col.ID],
			Covers:     covers[col.ID],
		})
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

// GetCollection 获取收藏夹详情（含图书列表，分页）
func GetCollection(c *gin.Context) {
	id := c.Param("id")
	uid := middleware.CurrentUserID(c)
	var col models.Collection
	if err := config.DB.Where("user_id = ?", uid).First(&col, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "collection not found"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	var total int64
	config.DB.Model(&models.CollectionBook{}).
		Where("collection_id = ? AND user_id = ?", col.ID, uid).
		Count(&total)

	var bookIDs []uint
	config.DB.Table("collection_books").
		Where("collection_id = ? AND user_id = ?", col.ID, uid).
		Order("created_at DESC").
		Offset((page-1)*pageSize).
		Limit(pageSize).
		Pluck("book_id", &bookIDs)

	var books []models.Book
	if len(bookIDs) > 0 {
		config.DB.Where("id IN ? AND user_id = ?", bookIDs, uid).Find(&books)
		// 按关联创建时间倒序排列
		order := map[uint]int{}
		for i, bid := range bookIDs {
			order[bid] = i
		}
		tmp := make([]models.Book, len(books))
		for _, b := range books {
			tmp[order[b.ID]] = b
		}
		ordered := make([]models.Book, 0, len(bookIDs))
		for _, b := range tmp {
			if b.ID != 0 {
				ordered = append(ordered, b)
			}
		}
		books = ordered
	}

	c.JSON(http.StatusOK, gin.H{
		"collection": col,
		"data":       books,
		"total":      total,
		"page":       page,
		"page_size":  pageSize,
	})
}

// CreateCollection 创建收藏夹（可选初始图书）
func CreateCollection(c *gin.Context) {
	var input struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		BookIDs     []uint `json:"book_ids"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name 不能为空"})
		return
	}

	uid := middleware.CurrentUserID(c)
	col := models.Collection{UserID: uid, Name: name, Description: input.Description}
	if err := config.DB.Create(&col).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if len(input.BookIDs) > 0 {
		addBooksToCollection(uid, col.ID, input.BookIDs)
	}
	c.JSON(http.StatusOK, col)
}

// UpdateCollection 更新收藏夹信息
func UpdateCollection(c *gin.Context) {
	id := c.Param("id")
	uid := middleware.CurrentUserID(c)
	var col models.Collection
	if err := config.DB.Where("user_id = ?", uid).First(&col, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "collection not found"})
		return
	}

	var input struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
		CoverPath   *string `json:"cover_path"`
		SortOrder   *int    `json:"sort_order"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name 不能为空"})
			return
		}
		updates["name"] = name
	}
	if input.Description != nil {
		updates["description"] = *input.Description
	}
	if input.CoverPath != nil {
		updates["cover_path"] = *input.CoverPath
	}
	if input.SortOrder != nil {
		updates["sort_order"] = *input.SortOrder
	}
	if len(updates) > 0 {
		config.DB.Model(&col).Updates(updates)
	}
	config.DB.First(&col, id)
	c.JSON(http.StatusOK, col)
}

// DeleteCollection 删除收藏夹（级联清理关联记录）
func DeleteCollection(c *gin.Context) {
	id := c.Param("id")
	uid := middleware.CurrentUserID(c)
	var col models.Collection
	if err := config.DB.Where("user_id = ?", uid).First(&col, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "collection not found"})
		return
	}
	config.DB.Where("collection_id = ? AND user_id = ?", col.ID, uid).Delete(&models.CollectionBook{})
	config.DB.Delete(&col)
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// AddBooksToCollection 向收藏夹添加图书
func AddBooksToCollection(c *gin.Context) {
	id := c.Param("id")
	uid := middleware.CurrentUserID(c)
	var col models.Collection
	if err := config.DB.Where("user_id = ?", uid).First(&col, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "collection not found"})
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
	added := addBooksToCollection(uid, col.ID, input.BookIDs)
	c.JSON(http.StatusOK, gin.H{"message": "added", "added": added})
}

// addBooksToCollection 批量关联图书（去重，仅限当前用户的书），返回实际新增数
func addBooksToCollection(uid, collectionID uint, bookIDs []uint) int64 {
	var valid []uint
	config.DB.Model(&models.Book{}).
		Where("id IN ? AND user_id = ?", bookIDs, uid).
		Pluck("id", &valid)

	var added int64
	for _, bid := range valid {
		var count int64
		config.DB.Model(&models.CollectionBook{}).
			Where("user_id = ? AND collection_id = ? AND book_id = ?", uid, collectionID, bid).
			Count(&count)
		if count == 0 {
			if err := config.DB.Create(&models.CollectionBook{
				UserID:       uid,
				CollectionID: collectionID,
				BookID:       bid,
			}).Error; err == nil {
				added++
			}
		}
	}
	return added
}

// RemoveBookFromCollection 从收藏夹移除图书
func RemoveBookFromCollection(c *gin.Context) {
	id := c.Param("id")
	bookID := c.Param("bookId")
	uid := middleware.CurrentUserID(c)
	res := config.DB.Where("user_id = ? AND collection_id = ? AND book_id = ?", uid, id, bookID).
		Delete(&models.CollectionBook{})
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "关联不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "removed"})
}

// GetBookCollections 获取图书所属的收藏夹
func GetBookCollections(c *gin.Context) {
	bookID := c.Param("id")
	uid := middleware.CurrentUserID(c)
	var book models.Book
	if err := config.DB.Where("user_id = ?", uid).First(&book, bookID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "book not found"})
		return
	}

	var collectionIDs []uint
	config.DB.Table("collection_books").
		Where("book_id = ? AND user_id = ?", bookID, uid).
		Pluck("collection_id", &collectionIDs)

	var collections []models.Collection
	if len(collectionIDs) > 0 {
		config.DB.Where("id IN ? AND user_id = ?", collectionIDs, uid).Find(&collections)
	}
	if collections == nil {
		collections = []models.Collection{}
	}
	c.JSON(http.StatusOK, gin.H{"data": collections})
}

// SetBookCollections 设置图书所属收藏夹（全量替换）
func SetBookCollections(c *gin.Context) {
	bookID := c.Param("id")
	uid := middleware.CurrentUserID(c)
	var book models.Book
	if err := config.DB.Where("user_id = ?", uid).First(&book, bookID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "book not found"})
		return
	}

	var input struct {
		CollectionIDs []uint `json:"collection_ids"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 全量替换：先清除原有关联，再建立新关联（仅限当前用户的收藏夹）
	config.DB.Where("book_id = ? AND user_id = ?", bookID, uid).Delete(&models.CollectionBook{})
	for _, cid := range input.CollectionIDs {
		var col models.Collection
		if err := config.DB.Where("user_id = ?", uid).First(&col, cid).Error; err == nil {
			config.DB.Create(&models.CollectionBook{
				UserID:       uid,
				CollectionID: cid,
				BookID:       book.ID,
			})
		}
	}

	var collections []models.Collection
	if len(input.CollectionIDs) > 0 {
		config.DB.Where("id IN ? AND user_id = ?", input.CollectionIDs, uid).Find(&collections)
	}
	if collections == nil {
		collections = []models.Collection{}
	}
	c.JSON(http.StatusOK, gin.H{"data": collections})
}
