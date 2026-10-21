package models

import "time"

// BookShare 图书分享表
type BookShare struct {
	ID        uint       `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID    uint       `json:"-" gorm:"not null;default:0;index"`  // 分享者
	BookID    uint       `json:"book_id" gorm:"not null;index"`      // 被分享的书
	Token     string     `json:"token" gorm:"size:64;uniqueIndex;not null"` // 分享码
	ExpiresAt *time.Time `json:"expires_at"`                          // 过期时间，nil 表示永久有效
	CreatedAt time.Time  `json:"created_at"`
}
