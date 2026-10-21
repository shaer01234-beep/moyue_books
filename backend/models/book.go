package models

import (
	"time"

	"gorm.io/gorm"
)

// Book 图书主表
type Book struct {
	ID          uint           `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID      uint           `json:"-" gorm:"not null;default:0;index"`
	Title       string         `json:"title" gorm:"size:512;not null;index"`
	Authors     string         `json:"authors" gorm:"type:text"`
	Description string         `json:"description" gorm:"type:text"`
	CoverPath   string         `json:"cover_path" gorm:"size:512"`
	FilePath    string         `json:"file_path" gorm:"size:512;not null"`
	FileType    string         `json:"file_type" gorm:"size:32;not null"`
	FileSize    int64          `json:"file_size"`
	Pages       int            `json:"pages"`
	Publisher   string         `json:"publisher" gorm:"size:256"`
	PublishDate string         `json:"publish_date" gorm:"size:64"`
	Language    string         `json:"language" gorm:"size:32"`
	ISBN        string         `json:"isbn" gorm:"size:64;index"`
	Tags        string         `json:"tags" gorm:"type:text"`
	Category    string         `json:"category" gorm:"size:128;index"`
	Series      string         `json:"series" gorm:"size:256"`
	SeriesIndex float64        `json:"series_index"`
	// 媒体库来源：LibraryID 非空表示该图书来自某个媒体库，FilePath 指向源文件原始路径（不迁移）。
	LibraryID      *uint  `json:"library_id" gorm:"index"`
	LibraryRelPath string `json:"library_rel_path" gorm:"size:1024"` // 相对媒体库根目录的路径（稳定去重键）
	SourceMissing  bool   `json:"source_missing" gorm:"default:false"` // 源文件是否已消失（被删除 / 移走）
	SourceModTime  int64  `json:"-" gorm:"default:0"`                  // 源文件修改时间（秒），用于变更检测
	Progress    float64        `json:"progress" gorm:"default:0"`
	CurrentPage int            `json:"current_page" gorm:"default:0"`
	LastRead    *time.Time     `json:"last_read"`
	TOC         string         `json:"-" gorm:"type:text"`            // 目录缓存 JSON（统一目录接口产物）
	TOCParsed   bool           `json:"-" gorm:"default:false"`        // 是否已尝试解析目录
	TTSLocation string         `json:"tts_location" gorm:"type:text"` // 朗读断点 JSON：{type,page|chapter,sent}
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

// Annotation 批注/划线表
type Annotation struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID    uint      `json:"-" gorm:"not null;default:0;index"`
	BookID    uint      `json:"book_id" gorm:"not null;index"`
	Type      string    `json:"type" gorm:"size:32;not null"` // highlight, underline, note
	Content   string    `json:"content" gorm:"type:text"`     // 选中的文本 / 笔记内容
	Note      string    `json:"note" gorm:"type:text"`        // 划线的批注笔记
	Position  string    `json:"position" gorm:"type:text"`    // 位置信息（JSON格式）
	Color     string    `json:"color" gorm:"size:32"`         // 划线颜色
	Style     string    `json:"style" gorm:"size:32"`         // 划线样式：highlight, underline, wavy, strikethrough
	Page      int       `json:"page"`
	Chapter   string    `json:"chapter" gorm:"size:512"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Bookmark 书签表
type Bookmark struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID    uint      `json:"-" gorm:"not null;default:0;index"`
	BookID    uint      `json:"book_id" gorm:"not null;index"`
	Page      int       `json:"page"`
	Position  string    `json:"position" gorm:"type:text"`
	Note      string    `json:"note" gorm:"type:text"`
	CreatedAt time.Time `json:"created_at"`
}

// Collection 收藏夹表
type Collection struct {
	ID          uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID      uint      `json:"-" gorm:"not null;default:0;index"`
	Name        string    `json:"name" gorm:"size:256;not null"`
	Description string    `json:"description" gorm:"type:text"`
	CoverPath   string    `json:"cover_path" gorm:"size:512"`
	SortOrder   int       `json:"sort_order" gorm:"default:0"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CollectionBook 收藏夹-图书关联表
type CollectionBook struct {
	UserID       uint      `json:"-" gorm:"not null;default:0;index"`
	CollectionID uint      `json:"collection_id" gorm:"primaryKey"`
	BookID       uint      `json:"book_id" gorm:"primaryKey"`
	CreatedAt    time.Time `json:"created_at"`
}

// BookReview 读后感表（读书笔记）
type BookReview struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID    uint      `json:"-" gorm:"not null;default:0;index"`
	BookID    uint      `json:"book_id" gorm:"not null;index"`
	Title     string    `json:"title" gorm:"size:256;not null"`
	Content   string    `json:"content" gorm:"type:text"` // Markdown 原文
	Rating    int       `json:"rating" gorm:"default:0"`  // 评分 0-5，0 表示未评分
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ReadingHistory 阅读历史表
type ReadingHistory struct {
	ID        uint       `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID    uint       `json:"-" gorm:"not null;default:0;index"`
	BookID    uint       `json:"book_id" gorm:"not null;index"`
	Type      string     `json:"type" gorm:"default:read;index"` // read=阅读, tts=朗读
	Engine    string     `json:"engine"`                        // 朗读引擎（edge-tts 等），仅 tts 记录
	StartPage int        `json:"start_page"`
	EndPage   int        `json:"end_page"`
	Duration  int        `json:"duration"` // 秒
	StartTime time.Time  `json:"start_time"`
	EndTime   *time.Time `json:"end_time"`
}

// ReadingPlan 阅读计划表
type ReadingPlan struct {
	ID           uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID       uint      `json:"-" gorm:"not null;default:0;index"`
	Name         string    `json:"name" gorm:"size:256;not null"`
	DailyGoalMin int       `json:"daily_goal_min" gorm:"not null;default:30"` // 每日目标（分钟）
	StartDate    string    `json:"start_date" gorm:"size:10"`                 // YYYY-MM-DD
	EndDate      string    `json:"end_date" gorm:"size:10"`
	BookID       *uint     `json:"book_id"` // 可选：指定图书
	Active       bool      `json:"active" gorm:"default:true"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Setting 设置表
type Setting struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID    uint      `json:"-" gorm:"not null;default:0;uniqueIndex:idx_user_key"`
	Key       string    `json:"key" gorm:"size:128;not null;uniqueIndex:idx_user_key"`
	Value     string    `json:"value" gorm:"type:text"`
	UpdatedAt time.Time `json:"updated_at"`
}

// BookWithCollections 用于查询的图书关联收藏夹结构
type BookWithCollections struct {
	Book
	Collections []Collection `json:"collections" gorm:"many2many:collection_books;"`
}
